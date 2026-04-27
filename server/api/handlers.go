package api

import (
	"net/http"
	"oncall-system/model"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetGroups(c *gin.Context) {
	var groups []model.Group
	if err := h.db.Order("`order` asc").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *Handler) GetPlatforms(c *gin.Context) {
	var platforms []model.Platform
	if err := h.db.Find(&platforms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *Handler) GetPeople(c *gin.Context) {
	var people []model.Person
	if err := h.db.Preload("Group").Find(&people).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, people)
}

func (h *Handler) GetShifts(c *gin.Context) {
	start := c.Query("start") // YYYY-MM-DD
	end := c.Query("end")     // YYYY-MM-DD

	var shifts []model.Shift
	query := h.db.Preload("Person").Preload("Group").Preload("Platform")

	if start != "" && end != "" {
		query = query.Where("date >= ? AND date <= ?", start, end)
	}

	if err := query.Find(&shifts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shifts)
}

func (h *Handler) BatchUpdatePeople(c *gin.Context) {
	var people []model.Person
	if err := c.ShouldBindJSON(&people); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := h.db.Begin()
	for _, p := range people {
		if p.GroupID == 0 {
			continue
		}
		var existing model.Person
		// Check by Name + Group
		if err := tx.Where("name = ? AND group_id = ?", p.Name, p.GroupID).First(&existing).Error; err == nil {
			// Update phone
			existing.Phone = p.Phone
			tx.Save(&existing)
		} else if err == gorm.ErrRecordNotFound {
			// Create new
			tx.Create(&p)
		} else {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	tx.Commit()
	c.Status(http.StatusOK)
}

func (h *Handler) DeletePerson(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	// Delete the person
	if err := h.db.Delete(&model.Person{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete person"})
		return
	}

	// Also delete related shifts
	h.db.Where("person_id = ?", id).Delete(&model.Shift{})

	c.JSON(http.StatusOK, gin.H{"message": "Person deleted successfully"})
}

type ShiftRequest struct {
	Date        string `json:"date"`
	GroupID     uint   `json:"group_id"`
	PlatformID  uint   `json:"platform_id"`
	PersonID    uint   `json:"person_id"`    // Optional single ID
	PersonIDs   []uint `json:"person_ids"`   // New: Multiple IDs
	PersonName  string `json:"person_name"`  // Optional create/find
	PersonPhone string `json:"person_phone"` // Optional create/find
}

func (h *Handler) UpsertShift(c *gin.Context) {
	var req ShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Resolve PersonIDs
	// If Name is provided, this logic supports creating *one* person.
	// For multiple selection, we rely on PersonIDs array.
	// We will combine single PersonID/Name logic (backward compat) with PersonIDs.

	targetPersonIDs := req.PersonIDs

	// Legacy single person support or create-new support
	if len(targetPersonIDs) == 0 {
		var singlePersonID uint = req.PersonID
		if req.PersonName != "" {
			var person model.Person
			switch err := h.db.Where("name = ? AND group_id = ?", req.PersonName, req.GroupID).First(&person).Error; err {
			case gorm.ErrRecordNotFound:
				newPerson := model.Person{
					Name:    req.PersonName,
					Phone:   req.PersonPhone,
					GroupID: req.GroupID,
				}
				if err := h.db.Create(&newPerson).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create person: " + err.Error()})
					return
				}
				singlePersonID = newPerson.ID
			case nil:
				// Update phone if needed
				if req.PersonPhone != "" && person.Phone != req.PersonPhone {
					person.Phone = req.PersonPhone
					h.db.Save(&person)
				}
				singlePersonID = person.ID
			}
		}
		if singlePersonID != 0 {
			targetPersonIDs = append(targetPersonIDs, singlePersonID)
		}
	}

	if len(targetPersonIDs) == 0 {
		// Valid case: clearing shifts
	}

	// 2. Transaction: Delete Old -> Insert New
	tx := h.db.Begin()

	// Delete existing shifts for this slot
	if err := tx.Where("date = ? AND group_id = ? AND platform_id = ?", req.Date, req.GroupID, req.PlatformID).Delete(&model.Shift{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear old shifts: " + err.Error()})
		return
	}

	// Insert new shifts
	var createdShifts []model.Shift
	for _, pid := range targetPersonIDs {
		newShift := model.Shift{
			Date:       req.Date,
			GroupID:    req.GroupID,
			PlatformID: req.PlatformID,
			PersonID:   pid,
		}
		if err := tx.Create(&newShift).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create shift: " + err.Error()})
			return
		}
		createdShifts = append(createdShifts, newShift)
	}

	tx.Commit()
	c.JSON(http.StatusOK, createdShifts)
}

func (h *Handler) GetTodayDashboard(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	// Fetch all groups and platforms to construct full grid
	var groups []model.Group
	h.db.Order("`order` asc").Find(&groups)

	var platforms []model.Platform
	h.db.Find(&platforms)

	var shifts []model.Shift
	h.db.Preload("Person").Preload("Group").Preload("Platform").Where("date = ?", today).Find(&shifts)

	// Map: GroupID -> PlatformID -> []Shift
	shiftMap := make(map[uint]map[uint][]model.Shift)
	for _, s := range shifts {
		if shiftMap[s.GroupID] == nil {
			shiftMap[s.GroupID] = make(map[uint][]model.Shift)
		}
		shiftMap[s.GroupID][s.PlatformID] = append(shiftMap[s.GroupID][s.PlatformID], s)
	}

	type PlatformShift struct {
		Platform model.Platform `json:"platform"`
		Shifts   []model.Shift  `json:"shifts"`
	}

	type DashboardItem struct {
		Group          model.Group     `json:"group"`
		PlatformShifts []PlatformShift `json:"platform_shifts"`
	}

	var result []DashboardItem
	for _, g := range groups {
		item := DashboardItem{Group: g}
		var pShifts []PlatformShift

		// 根据组类型决定显示哪些平台
		var displayPlatforms []model.Platform
		switch g.Name {
		case "运维":
			// 运维组：只显示 primary/backup 平台
			for _, p := range platforms {
				if p.SubType == "primary" || p.SubType == "backup" {
					displayPlatforms = append(displayPlatforms, p)
				}
			}
		case "后台":
			// 后台组：只显示 web/数仓 平台
			for _, p := range platforms {
				if p.SubType == "web" || p.SubType == "数仓" {
					displayPlatforms = append(displayPlatforms, p)
				}
			}
		default:
			// 其他组：只显示 server/client 平台
			for _, p := range platforms {
				if p.SubType == "server" || p.SubType == "client" {
					displayPlatforms = append(displayPlatforms, p)
				}
			}
		}

		for _, p := range displayPlatforms {
			ps := PlatformShift{Platform: p}
			if groupMap, ok := shiftMap[g.ID]; ok {
				if shifts, ok := groupMap[p.ID]; ok {
					ps.Shifts = shifts
				}
			}
			pShifts = append(pShifts, ps)
		}
		item.PlatformShifts = pShifts
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}
