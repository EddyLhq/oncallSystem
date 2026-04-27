package main

import (
	"log"
	"oncall-system/api"
	"oncall-system/model"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize Database
	db, err := gorm.Open(sqlite.Open("oncall.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// Migrate Schema
	err = model.Migrate(db)
	if err != nil {
		log.Fatal("failed to migrate database")
	}

	// Seed data if empty (Optional, strictly for demo purposes)
	seedData(db)

	// Initialize Router
	r := gin.Default()

	// CORS Setup
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all for dev
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API Routes
	apiGroup := r.Group("/api")
	{
		handler := api.NewHandler(db)
		apiGroup.GET("/groups", handler.GetGroups)
		apiGroup.GET("/people", handler.GetPeople)
		apiGroup.POST("/people/batch", handler.BatchUpdatePeople)
		apiGroup.DELETE("/people/:id", handler.DeletePerson)
		apiGroup.GET("/platforms", handler.GetPlatforms)
		apiGroup.GET("/shifts", handler.GetShifts)
		apiGroup.POST("/shifts", handler.UpsertShift)
		apiGroup.GET("/dashboard/today", handler.GetTodayDashboard)
	}

	r.Run(":8888")
}

func seedData(db *gorm.DB) {
	// Seed Groups
	var groupCount int64
	db.Model(&model.Group{}).Count(&groupCount)
	if groupCount == 0 {
		groups := []model.Group{
			{Name: "运维", Order: 1},
			{Name: "大厅", Order: 2},
			{Name: "新娱乐", Order: 3},
			{Name: "本地", Order: 4},
			{Name: "公共", Order: 5},
			{Name: "国际", Order: 6},
		}
		db.Create(&groups)

		// Add dummy people
		people := []model.Person{}
		for _, g := range groups {
			people = append(people, model.Person{Name: g.Name + "-人员A", Phone: "13800138000", GroupID: g.ID})
			people = append(people, model.Person{Name: g.Name + "-人员B", Phone: "13900139000", GroupID: g.ID})
		}
		db.Create(&people)
	}

	// Seed Platforms
	var platformCount int64
	db.Model(&model.Platform{}).Count(&platformCount)
	if platformCount == 0 {
		platforms := []model.Platform{
			// 运维组使用 primary/backup
			{Name: "Idn", SubType: "primary", ID: 1},
			{Name: "Idn", SubType: "backup", ID: 2},
			{Name: "Idn-Sub", SubType: "primary", ID: 3},
			{Name: "Idn-Sub", SubType: "backup", ID: 4},
			{Name: "Malaysia", SubType: "primary", ID: 5},
			{Name: "Malaysia", SubType: "backup", ID: 6},
			// 其他组使用 server/client
			{Name: "Idn", SubType: "server", ID: 7},
			{Name: "Idn", SubType: "client", ID: 8},
			{Name: "Idn-Sub", SubType: "server", ID: 9},
			{Name: "Idn-Sub", SubType: "client", ID: 10},
			{Name: "Malaysia", SubType: "server", ID: 11},
			{Name: "Malaysia", SubType: "client", ID: 12},
		}
		db.Create(&platforms)
	}

	// Ensure `后台` group and platforms exist if added later
	var backGroupCount int64
	db.Model(&model.Group{}).Where("name = ?", "后台").Count(&backGroupCount)
	if backGroupCount == 0 {
		backGroup := model.Group{Name: "后台", Order: 7}
		db.Create(&backGroup)
		people := []model.Person{
			{Name: "后台-人员A", Phone: "13800138001", GroupID: backGroup.ID},
			{Name: "后台-人员B", Phone: "13900139002", GroupID: backGroup.ID},
		}
		db.Create(&people)
	}

	var backPlatformCount int64
	db.Model(&model.Platform{}).Where("sub_type = ?", "web").Count(&backPlatformCount)
	if backPlatformCount == 0 {
		backPlatforms := []model.Platform{
			// 后台组使用 web/数仓
			{Name: "Idn", SubType: "web"},
			{Name: "Idn", SubType: "数仓"},
			{Name: "Idn-Sub", SubType: "web"},
			{Name: "Idn-Sub", SubType: "数仓"},
			{Name: "Malaysia", SubType: "web"},
			{Name: "Malaysia", SubType: "数仓"},
		}
		db.Create(&backPlatforms)
	}
}
