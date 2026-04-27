<script setup lang="ts">
import { ref, onMounted, onUnmounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import dayjs from 'dayjs'
import { Refresh } from '@element-plus/icons-vue'

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

interface PlatformShift {
  platform: Platform
  shifts: {
    id: number
    person: {
      name: string
      phone: string
    }
  }[]
}

interface DashboardItem {
  group: Group
  platform_shifts: PlatformShift[]
}

const displayData = ref<DashboardItem[]>([])
const groups = ref<Group[]>([])
const platforms = ref<Platform[]>([])
const currentTime = ref(dayjs().format('HH:mm:ss'))
const currentDateDesc = ref(dayjs().format('YYYY年MM月DD日 dddd'))
const calendarDays = ref<any[]>([])
const isLiveMode = ref(true)
const selectedDateStr = ref(dayjs().format('YYYY-MM-DD'))

// Helper function to get platform display name
const getPlatformDisplayName = (platform: Platform) => {
  if (!platform.sub_type) {
    return platform.name
  }
  if (platform.sub_type === 'primary') {
    return '主'
  }
  if (platform.sub_type === 'backup') {
    return '备'
  }
  if (platform.sub_type === 'server') {
    return '服务端'
  }
  if (platform.sub_type === 'client') {
    return '客户端'
  }
  if (platform.sub_type === 'web') {
    return 'web'
  }
  if (platform.sub_type === '数仓') {
    return '数仓'
  }
  return platform.name
}

// Get unique platform names from platform_shifts
const getUniquePlatformNames = (platformShifts: PlatformShift[]) => {
  const names = new Set<string>()
  platformShifts.forEach(ps => names.add(ps.platform.name))
  return Array.from(names)
}

// Get platforms by name
const getPlatformsByName = (platformShifts: PlatformShift[], name: string) => {
  return platformShifts.filter(ps => ps.platform.name === name)
}

// Fetch all groups and platforms once
const fetchMetadata = async () => {
    try {
        const [gRes, pRes] = await Promise.all([
             axios.get('/api/groups'),
             axios.get('/api/platforms')
        ])
        groups.value = gRes.data
        platforms.value = pRes.data
    } catch (error) {
        console.error("Failed to fetch metadata", error)
    }
}

const fetchToday = async () => {
  if (!isLiveMode.value) return // Don't overwrite if viewing history
  
  try {
    const res = await axios.get('/api/dashboard/today')
    displayData.value = res.data
    selectedDateStr.value = dayjs().format('YYYY-MM-DD')
    updateDateDesc(dayjs())
  } catch (error) {
    console.error(error)
  }
}

const fetchMonthPreview = async () => {
  const start = dayjs().startOf('month').format('YYYY-MM-DD')
  const end = dayjs().endOf('month').format('YYYY-MM-DD')
  console.log('📅 获取月度预览:', start, '到', end)
  try {
    const res = await axios.get(`/api/shifts?start=${start}&end=${end}`)
    const shifts = res.data // array of shifts
    console.log('✅ 获取到', shifts.length, '条排班记录')
    
    // Group shifts by date
    // We need to render a calendar grid
    const days = []
    const daysInMonth = dayjs().daysInMonth()
    const startObj = dayjs().startOf('month')
    
    let daysWithShifts = 0
    
    // Simple grid: just days of month
    for (let i = 0; i < daysInMonth; i++) {
      const d = startObj.add(i, 'day')
      const dateStr = d.format('YYYY-MM-DD')
      
      // Find shifts for this day
      // 注意：后端返回的date是ISO格式（如"2026-02-11T00:00:00Z"），需要格式化后比较
      const dayShifts = shifts.filter((s: any) => {
        const shiftDate = dayjs(s.date).format('YYYY-MM-DD')
        return shiftDate === dateStr
      })
      
      days.push({
        date: d.date(),
        fullDate: dateStr,
        hasShift: dayShifts.length > 0,
        shifts: dayShifts
      })
      
      if (dayShifts.length > 0) {
        daysWithShifts++
      }
    }
    calendarDays.value = days
    console.log('📊 日历更新完成:', daysWithShifts, '天有排班数据')
  } catch (error) {
    console.error('❌ 获取月度预览失败:', error)
  }
}

const handleDateClick = (day: any) => {
    isLiveMode.value = false
    selectedDateStr.value = day.fullDate
    updateDateDesc(dayjs(day.fullDate))
    
    console.log('=== 点击日期 ===', day.fullDate)
    console.log('找到shifts数量:', day.shifts?.length || 0)
    if (day.shifts && day.shifts.length > 0) {
        console.log('第一个shift示例:', day.shifts[0])
    }
    
    // Construct display data locally using groups, platforms and day.shifts
    const newDisplayData: DashboardItem[] = []
    
    // Map: GroupID -> PlatformID -> []Shift
    const shiftMap = new Map()
    if (day.shifts && Array.isArray(day.shifts)) {
        day.shifts.forEach((s: any) => {
            if (!shiftMap.has(s.group_id)) shiftMap.set(s.group_id, new Map())
            const groupMap = shiftMap.get(s.group_id)
            if (!groupMap.has(s.platform_id)) groupMap.set(s.platform_id, [])
            groupMap.get(s.platform_id).push(s)
        })
    }

    groups.value.forEach(g => {
        const pShifts: PlatformShift[] = []
        
        // 根据组类型决定显示哪些平台
        let displayPlatforms: Platform[] = []
        if (g.name === '运维') {
            // 运维组：只显示 primary/backup 平台
            displayPlatforms = platforms.value.filter(p => p.sub_type === 'primary' || p.sub_type === 'backup')
        } else if (g.name === '后台') {
            // 后台组：只显示 web/数仓 平台
            displayPlatforms = platforms.value.filter(p => p.sub_type === 'web' || p.sub_type === '数仓')
        } else {
            // 其他组：只显示 server/client 平台
            displayPlatforms = platforms.value.filter(p => p.sub_type === 'server' || p.sub_type === 'client')
        }
        
        displayPlatforms.forEach(p => {
            const shifts = shiftMap.get(g.id)?.get(p.id) || []
            pShifts.push({
                platform: p,
                shifts: shifts
            })
        })
        
        newDisplayData.push({
            group: g,
            platform_shifts: pShifts
        })
    })
    
    console.log('生成的组数量:', newDisplayData.length)
    console.log('==================')
    displayData.value = newDisplayData
}

const returnToLive = () => {
    isLiveMode.value = true
    fetchToday()
}

const updateDateDesc = (d: dayjs.Dayjs) => {
    currentDateDesc.value = d.format('YYYY年MM月DD日 dddd')
}

const router = useRouter()
let lastRoute = ref('')
let lastRefreshTime = 0
const MIN_REFRESH_INTERVAL = 2000 // 2秒内不重复刷新

// 带防抖的刷新函数
const refreshData = () => {
  const now = Date.now()
  if (now - lastRefreshTime < MIN_REFRESH_INTERVAL) {
    console.log('刷新请求被防抖，距离上次刷新不足2秒')
    return
  }
  lastRefreshTime = now
  console.log('执行数据刷新')
  fetchToday()
  fetchMonthPreview()
}

let timer: number
let refreshCounter = 0

onMounted(async () => {
  await fetchMetadata()
  fetchToday()
  fetchMonthPreview()
  lastRefreshTime = Date.now() // 记录初始刷新时间
  
  // 记录当前路由
  lastRoute.value = router.currentRoute.value.name as string
  
  timer = setInterval(() => {
    currentTime.value = dayjs().format('HH:mm:ss')
    refreshCounter++
    
    // 每秒检查是否需要刷新今日数据
    if (new Date().getSeconds() === 0 && isLiveMode.value) {
      fetchToday()
    }
    
    // 每10秒刷新一次月度预览数据（从30秒改为10秒，更即时）
    if (refreshCounter % 10 === 0) {
      console.log('定时刷新月度预览数据')
      fetchMonthPreview()
      lastRefreshTime = Date.now() // 更新刷新时间
    }
  }, 1000)
})

// 当组件被激活时（从其他页面返回）
onActivated(() => {
  console.log('Dashboard组件被激活')
  refreshData()
})

// 监听路由变化
router.afterEach((to, from) => {
  if (to.name === 'dashboard' && from.name === 'admin') {
    console.log('从Admin返回Dashboard')
    refreshData()
  }
})

onUnmounted(() => {
  clearInterval(timer)
})
</script>

<template>
  <div class="dashboard-container">
    <div class="dashboard-header">
      <div class="title">松壳应急值班大屏</div>
      <div class="center-status" v-if="!isLiveMode">
        <span class="preview-tag">预览模式</span>
        <el-button type="primary" link @click="returnToLive">
          <el-icon><Refresh /></el-icon> 返回今日实时
        </el-button>
      </div>
      <div class="time-box">
        <div class="date">{{ currentDateDesc }}</div>
        <div class="time">{{ currentTime }}</div>
      </div>
    </div>

    <div class="content">
      <div class="main-panel">
        <h2 class="section-title">
            {{ isLiveMode ? '今日值班人员' : '值班人员 (预览)' }}
        </h2>
        <div class="cards-container">
          <div v-for="(item, index) in displayData" :key="index" class="duty-card glass">
            <div class="card-header">
              <div class="group-name">{{ item.group.name }}</div>
              <div class="sub-type-headers">
                <span v-for="ps in item.platform_shifts.slice(0, (item.group.name === '运维' || item.group.name === '后台') ? 2 : 2)" 
                      :key="ps.platform.id" 
                      class="sub-type-label">
                  {{ getPlatformDisplayName(ps.platform) }}
                </span>
              </div>
            </div>
            <div class="platforms-table">
              <div v-for="platformName in getUniquePlatformNames(item.platform_shifts)" 
                   :key="platformName" 
                   class="platform-row">
                <span class="platform-label">{{ platformName }}:</span>
                <div class="platform-cells">
                  <div v-for="ps in getPlatformsByName(item.platform_shifts, platformName)" 
                       :key="ps.platform.id" 
                       class="platform-cell">
                    <div class="persons-container" v-if="ps.shifts && ps.shifts.length > 0">
                      <span v-for="shift in ps.shifts" :key="shift.id" class="person-info">
                        <span class="name">{{ shift.person.name }}</span>
                        <a :href="`tel:${shift.person.phone}`" class="phone">{{ shift.person.phone }}</a>
                      </span>
                    </div>
                    <span v-else class="empty">待定</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="side-panel glass">
        <h3 class="section-title small">本月排班预览</h3>
        <div class="month-grid">
          <div 
            v-for="day in calendarDays" 
            :key="day.fullDate" 
            class="day-cell"
            :class="{ 
                'has-shift': day.hasShift, 
                'is-today': day.fullDate === dayjs().format('YYYY-MM-DD'),
                'is-selected': day.fullDate === selectedDateStr
            }"
            @click="handleDateClick(day)"
          >
            <span class="day-number">{{ day.date }}</span>
            <div v-if="day.hasShift" class="shift-indicator">
              <span class="dot"></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Orbitron:wght@400;700&display=swap');

.dashboard-container {
  width: 100vw;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  font-family: 'Inter', sans-serif;
}

.dashboard-header {
  padding: 20px 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid rgba(255,255,255,0.1);
  background: rgba(0,0,0,0.2);
}

.title {
  font-size: 32px;
  font-weight: bold;
  letter-spacing: 2px;
  background: linear-gradient(90deg, #60a5fa, #a78bfa);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.center-status {
    display: flex;
    align-items: center;
    gap: 15px;
}

.preview-tag {
    background: #eab308;
    color: #000;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 14px;
    font-weight: bold;
}

.time-box {
  text-align: right;
}

.time {
  font-size: 36px;
  font-family: 'Orbitron', monospace;
  font-weight: 700;
  color: #38bdf8;
}

.date {
  font-size: 14px;
  color: #94a3b8;
}

.content {
  flex: 1;
  display: flex;
  padding: 40px;
  gap: 40px;
}

.main-panel {
  flex: 2;
  display: flex;
  flex-direction: column;
}

.side-panel {
  flex: 1;
  padding: 20px;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
}

.section-title {
  font-size: 24px;
  margin-bottom: 30px;
  color: #e2e8f0;
  border-left: 4px solid #3b82f6;
  padding-left: 15px;
}
.section-title.small {
  font-size: 18px;
  margin-bottom: 20px;
}

.cards-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
  flex: 1;
  overflow-y: auto;
}

.duty-card {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 16px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  position: relative;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  border: 1px solid rgba(255,255,255,0.05);
}

.duty-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 30px rgba(0,0,0,0.5);
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(59, 130, 246, 0.5);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
  padding-bottom: 10px;
}

.group-name {
  font-size: 18px;
  color: #60a5fa;
  text-transform: uppercase;
  letter-spacing: 1px;
  font-weight: bold;
}

.sub-type-headers {
  display: flex;
  gap: 80px;
}

.sub-type-label {
  color: #94a3b8;
  font-size: 14px;
  font-weight: bold;
  min-width: 80px;
  text-align: center;
}

.platforms-table {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.platform-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.platform-label {
  color: #94a3b8;
  font-size: 14px;
  width: 100px;
  padding-top: 6px;
  flex-shrink: 0;
}

.platform-cells {
  display: flex;
  gap: 20px;
  flex: 1;
}

.platform-cell {
  flex: 1;
  min-width: 80px;
}

.persons-container {
    display: flex;
    flex-direction: column;
    gap: 5px;
    flex: 1;
}

.person-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: rgba(255,255,255,0.05);
  padding: 8px 10px;
  border-radius: 6px;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

.name {
  color: #fff;
  font-weight: bold;
  font-size: 16px;
  word-break: break-word;
}

.phone {
  color: #10b981;
  font-size: 13px;
  font-family: monospace;
  white-space: nowrap;
  letter-spacing: -0.5px;
  text-decoration: none;
}

.phone:hover {
  text-decoration: underline;
  opacity: 0.8;
}

.empty {
  color: #475569;
  font-style: italic;
  font-size: 14px;
  padding-top: 6px;
}

/* Glassmorphism */
.glass {
  background: rgba(255, 255, 255, 0.03);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

/* Calendar Grid */
.month-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 8px;
  flex: 1;
}

.day-cell {
  background: rgba(255,255,255,0.02);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  min-height: 50px;
  position: relative;
  cursor: pointer;
  transition: all 0.2s;
}

.day-cell:hover {
    background: rgba(255,255,255,0.1);
}

.day-cell.is-today {
  border: 1px solid #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.day-cell.is-selected {
    background: rgba(16, 185, 129, 0.2);
    border: 1px solid #10b981;
}

.day-number {
  font-size: 14px;
  color: #64748b;
}

.has-shift .day-number {
  color: #e2e8f0;
  font-weight: bold;
}

.shift-indicator {
  display: flex;
  justify-content: center;
  margin-top: 4px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 4px #10b981;
}
</style>
