<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import axios from 'axios'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Group {
  id: number
  name: string
  order: number
}

interface Platform {
  id: number
  name: string
  sub_type: string
}

interface Person {
  id: number
  name: string
  phone: string
  group_id: number
}

interface Shift {
  id: number
  date: string
  group_id: number
  platform_id: number
  person_id: number
}

const currentDate = ref(new Date())
const groups = ref<Group[]>([])
const platforms = ref<Platform[]>([])
const people = ref<Person[]>([])
const shifts = ref<Shift[]>([])

// Scheduling Dialog State
const dialogVisible = ref(false)
const editingDate = ref('')
const editingDateTitle = ref('')
const isSaving = ref(false)

// Person Management Dialog State
const personDialogVisible = ref(false)
const activeGroupTab = ref('0') 
const batchInputText = ref('')

// Form data: Key = `${groupId}-${platformId}`
// Value: personId (number) or personIds (number[])
const currentAssignment = ref<Record<string, any>>({})

// Helper function to get platform display name
const getPlatformDisplayName = (platform: Platform) => {
  if (!platform.sub_type) {
    return platform.name
  }
  if (platform.sub_type === 'primary') {
    return platform.name + '-主'
  }
  if (platform.sub_type === 'backup') {
    return platform.name + '-备'
  }
  if (platform.sub_type === 'server') {
    return platform.name + '-服务端'
  }
  if (platform.sub_type === 'client') {
    return platform.name + '-客户端'
  }
  if (platform.sub_type === 'web') {
    return platform.name + '-web'
  }
  if (platform.sub_type === '数仓') {
    return platform.name + '-数仓'
  }
  return platform.name
}

// Helper function to get platforms for a specific group
// 运维组: 只显示 primary/backup 平台
// 其他组: 只显示 server/client 平台
const getPlatformsForGroup = (group: Group) => {
  if (group.name === '运维') {
    return platforms.value.filter(p => p.sub_type === 'primary' || p.sub_type === 'backup')
  } else if (group.name === '后台') {
    return platforms.value.filter(p => p.sub_type === 'web' || p.sub_type === '数仓')
  } else {
    return platforms.value.filter(p => p.sub_type === 'server' || p.sub_type === 'client')
  }
}

const getPeopleByGroup = (groupId: number) => {
  return people.value.filter(p => p.group_id === groupId)
}

const fetchData = async () => {
  try {
    const [groupsRes, peopleRes, platformsRes] = await Promise.all([
      axios.get('/api/groups'),
      axios.get('/api/people'),
      axios.get('/api/platforms')
    ])
    groups.value = groupsRes.data
    people.value = peopleRes.data
    platforms.value = platformsRes.data
    
    // Set default tab for person management
    if (groups.value.length > 0 && groups.value[0]) {
        activeGroupTab.value = groups.value[0].id.toString()
    }

    // Fetch shifts for current month
    const start = dayjs(currentDate.value).startOf('month').format('YYYY-MM-DD')
    const end = dayjs(currentDate.value).endOf('month').format('YYYY-MM-DD')
    await fetchShifts(start, end)
  } catch (error) {
    ElMessage.error('Failed to load data')
  }
}

const fetchShifts = async (start: string, end: string) => {
  try {
    const res = await axios.get(`/api/shifts?start=${start}&end=${end}`)
    shifts.value = res.data
  } catch (error) {
    ElMessage.error('Failed to load shifts')
  }
}

const openEditDialog = (date: Date) => {
    editingDate.value = dayjs(date).format('YYYY-MM-DD')
    editingDateTitle.value = dayjs(date).format('YYYY年MM月DD日 排班设置')
    
    // Initialize form data
    const assignment: Record<string, any> = {}
    
    groups.value.forEach(g => {
        platforms.value.forEach(p => {
             const key = `${g.id}-${p.id}`
             
             // Find all shifts for this slot
             const slotShifts = shifts.value.filter(s => 
                s.date === editingDate.value && 
                s.group_id === g.id && 
                s.platform_id === p.id
             )
             
             if (slotShifts.length > 0) {
                 // 所有组都是单选，取第一个ID
                 assignment[key] = slotShifts[0]!.person_id
             } else {
                 // Default empty value
                 assignment[key] = undefined
             }
        })
    })
    
    currentAssignment.value = assignment
    dialogVisible.value = true
}

const saveAllShifts = async () => {
    isSaving.value = true
    try {
        const promises: Promise<any>[] = []
        
        for (const g of groups.value) {
            for (const p of getPlatformsForGroup(g)) {
                const key = `${g.id}-${p.id}`
                const value = currentAssignment.value[key]
                
                // 只有当用户明确选择了人员时才发送请求
                // undefined 表示用户没有操作这个slot，保持原样
                if (value !== undefined && value !== null && value !== '') {
                    let personIDs: number[] = []
                    
                    if (Array.isArray(value)) {
                        personIDs = value.filter(id => id > 0)
                    } else if (typeof value === 'number' && value > 0) {
                        personIDs = [value]
                    }
                    
                    // 只有当有有效的人员ID时才发送请求
                    if (personIDs.length > 0) {
                        const req = axios.post('/api/shifts', {
                            date: editingDate.value,
                            group_id: g.id,
                            platform_id: p.id,
                            person_ids: personIDs
                        })
                        promises.push(req)
                    }
                }
            }
        }
        
        await Promise.all(promises)
        
        // Refresh data
        const start = dayjs(currentDate.value).startOf('month').format('YYYY-MM-DD')
        const end = dayjs(currentDate.value).endOf('month').format('YYYY-MM-DD')
        await fetchShifts(start, end)
        
        ElMessage.success('保存成功')
        dialogVisible.value = false
        
    } catch (error) {
        console.error(error)
        ElMessage.error('保存失败')
    } finally {
        isSaving.value = false
    }
}

// Person Management Logic
const openPersonDialog = () => {
    personDialogVisible.value = true
    batchInputText.value = ''
}

const saveBatchPeople = async () => {
    const groupId = parseInt(activeGroupTab.value)
    if (!groupId) return

    // Parse text area: "Name Phone" per line
    const lines = batchInputText.value.split('\n')
    const peopleToSave: any[] = []

    lines.forEach(line => {
        const parts = line.trim().split(/\s+/)
        if (parts.length >= 2) {
            peopleToSave.push({
                name: parts[0],
                phone: parts[1],
                group_id: groupId
            })
        }
    })

    if (peopleToSave.length === 0) {
        ElMessage.warning('请输入有效的姓名和电话')
        return
    }

    try {
        await axios.post('/api/people/batch', peopleToSave)
        ElMessage.success('批量更新成功')
        
        // Refresh people list
        const peopleRes = await axios.get('/api/people')
        people.value = peopleRes.data
        
        batchInputText.value = '' // clear input
    } catch (error) {
        ElMessage.error('更新失败')
    }
}

const deletePerson = async (personId: number) => {
    try {
        await ElMessageBox.confirm('确定要删除这个人员吗？删除后相关的排班记录也会被删除。', '确认删除', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        })
        
        await axios.delete(`/api/people/${personId}`)
        ElMessage.success('删除成功')
        
        // Refresh people list
        const peopleRes = await axios.get('/api/people')
        people.value = peopleRes.data
        
        // Refresh shifts
        const start = dayjs(currentDate.value).startOf('month').format('YYYY-MM-DD')
        const end = dayjs(currentDate.value).endOf('month').format('YYYY-MM-DD')
        await fetchShifts(start, end)
    } catch (error: any) {
        if (error !== 'cancel') {
            ElMessage.error('删除失败')
        }
    }
}

const getPersonPhone = (value: number | number[]) => {
    if (Array.isArray(value)) {
        // Multi-select: Join phones
        return value.map(id => {
            const p = people.value.find(per => per.id === id)
            return p ? p.phone : ''
        }).filter(p => p).join(', ')
    } else {
        // Single select
        const p = people.value.find(per => per.id === value)
    return p ? p.phone : ''
    }
}


// Calculate shift count for calendar preview
const getShiftCountForDate = (date: string) => {
    return shifts.value.filter(s => s.date === date).length
}

onMounted(() => {
  fetchData()
})

watch(currentDate, (newDate) => {
    const start = dayjs(newDate).startOf('month').format('YYYY-MM-DD')
    const end = dayjs(newDate).endOf('month').format('YYYY-MM-DD')
    fetchShifts(start, end)
})

</script>

<template>
  <div class="admin-container">
    <div class="header">
      <h2>排班管理后台</h2>
      <div class="header-actions">
           <el-button type="success" @click="openPersonDialog">人员管理</el-button>
           <el-button type="primary" @click="$router.push('/dashboard')">查看大屏</el-button>
      </div>
    </div>

    <el-calendar v-model="currentDate">
      <template #date-cell="{ data }">
        <div class="date-cell" @click.stop="openEditDialog(data.date)">
          <div class="date-label">{{ data.day.split('-').slice(2).join('') }}</div>
          <div class="cell-content">
             <div class="status-summary" v-if="getShiftCountForDate(data.day) > 0">
                 已安排 {{ getShiftCountForDate(data.day) }} 人
             </div>
             <el-button size="small" type="primary" link class="edit-btn">点击编辑</el-button>
          </div>
        </div>
      </template>
    </el-calendar>

    <!-- Schedule Editing Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingDateTitle" width="90%" top="5vh">
        <div class="edit-grid">
            <div v-for="group in groups" :key="group.id" class="group-section">
                <div class="dialog-group-name">{{ group.name }}</div>
                <div class="platforms-container">
                    <div v-for="platform in getPlatformsForGroup(group)" :key="platform.id" class="platform-item">
                        <span class="platform-name">{{ getPlatformDisplayName(platform) }}</span>
                        <div class="select-group">
                             <el-select 
                                v-model="currentAssignment[`${group.id}-${platform.id}`]" 
                                placeholder="选择人员"
                                size="small"
                                style="width: 140px"
                                clearable
                                :multiple="false"
                                :multiple-limit="0"
                                collapse-tags
                             >
                                <el-option 
                                    v-for="p in getPeopleByGroup(group.id)"
                                    :key="p.id"
                                    :label="p.name"
                                    :value="p.id"
                                />
                             </el-select>
                             <span class="phone-display">
                                 {{ getPersonPhone(currentAssignment[`${group.id}-${platform.id}`] || 0) }}
                             </span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <template #footer>
            <span class="dialog-footer">
                <el-button @click="dialogVisible = false">取消</el-button>
                <el-button type="primary" :loading="isSaving" @click="saveAllShifts">
                    全部保存
                </el-button>
            </span>
        </template>
    </el-dialog>

    <!-- Person Management Dialog -->
    <el-dialog v-model="personDialogVisible" title="人员管理 (批量录入)" width="600px">
        <el-tabs v-model="activeGroupTab" type="card">
            <el-tab-pane v-for="group in groups" :key="group.id" :label="group.name" :name="group.id.toString()">
                <div class="person-mgmt-content">
                    <div class="current-list">
                        <h4>当前人员列表</h4>
                        <el-table :data="getPeopleByGroup(group.id)" height="200" style="width: 100%">
                            <el-table-column prop="name" label="姓名" width="100" />
                            <el-table-column prop="phone" label="电话" />
                            <el-table-column label="操作" width="60" align="center">
                                <template #default="scope">
                                    <el-button 
                                        type="danger" 
                                        link 
                                        size="small"
                                        @click="deletePerson(scope.row.id)"
                                    >
                                        ✕
                                    </el-button>
                                </template>
                            </el-table-column>
                        </el-table>
                    </div>
                    <div class="batch-input-area">
                        <h4>批量录入/更新</h4>
                        <p class="hint">格式：姓名 电话 (每行一个)</p>
                        <el-input
                            v-model="batchInputText"
                            type="textarea"
                            :rows="5"
                            placeholder="例如：
张三 13800138000
李四 13900139000"
                        />
                        <div class="batch-actions">
                            <el-button type="primary" @click="saveBatchPeople">确认更新当前组</el-button>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
        </el-tabs>
    </el-dialog>
  </div>
</template>

<style scoped>
.admin-container {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: 100vh;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.date-cell {
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
  cursor: pointer;
  transition: background-color 0.2s;
}
.date-cell:hover {
    background-color: #ecf5ff;
}
.date-label {
  font-weight: bold;
}
.cell-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 5px;
}
.status-summary {
    font-size: 12px;
    color: #67c23a;
    background: #f0f9eb;
    padding: 2px 5px;
    border-radius: 4px;
}

/* Edit Dialog Styles */
.edit-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 15px;
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 5px;
}
.group-section {
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    padding: 12px;
    background: #fff;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
}
.dialog-group-name {
    font-weight: bold;
    font-size: 16px;
    color: #409eff;
    margin-bottom: 10px;
    border-bottom: 1px solid #EBEEF5;
    padding-bottom: 8px;
}
.platforms-container {
    display: flex;
    flex-direction: column;
    gap: 10px;
}
.platform-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
}
.platform-name {
    font-size: 13px;
    color: #606266;
    width: 60px;
    flex-shrink: 0;
}
.select-group {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
}
.phone-display {
    font-size: 13px;
    color: #909399;
}
.header-actions {
    display: flex;
    gap: 15px;
}
/* Person Management Styles */
.person-mgmt-content {
    display: flex;
    gap: 20px;
}
.current-list {
    flex: 1;
    border-right: 1px solid #EBEEF5;
    padding-right: 20px;
}
.batch-input-area {
    flex: 1;
}
.hint {
    font-size: 12px;
    color: #909399;
    margin-bottom: 10px;
}
.batch-actions {
    margin-top: 15px;
    text-align: right;
}
</style>
