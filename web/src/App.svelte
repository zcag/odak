<script>
  import { onMount, onDestroy, tick } from 'svelte'
  import { fade, scale, fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'

  /* ── theme definitions ────────────────────────────── */
  const DARK_THEMES = {
    Void: {
      '--bg':'#080808','--bg-side':'#0b0b0b','--bg-hover':'#0e0e0e','--bg-card':'#0f0f0f','--bg-pop':'#131313',
      '--bd':'#181818','--bd2':'#202020','--bd3':'#2a2a2a',
      '--tx':'#c8c8c8','--tx2':'#606060','--tx3':'#3a3a3a','--tx4':'#262626',
      '--tx-done':'#282828','--tx-head':'#e0e0e0',
      '--accent':'#6366f1','--accent-bg':'rgba(99,102,241,.07)','--accent-glow':'rgba(99,102,241,.45)',
      '--del-hover':'#f87171','--sync-fresh':'#34d399'
    },
    Carbon: {
      '--bg':'#111111','--bg-side':'#141414','--bg-hover':'#181818','--bg-card':'#161616','--bg-pop':'#1c1c1c',
      '--bd':'#222222','--bd2':'#2c2c2c','--bd3':'#383838',
      '--tx':'#d0d0d0','--tx2':'#6a6a6a','--tx3':'#484848','--tx4':'#323232',
      '--tx-done':'#303030','--tx-head':'#eaeaea',
      '--accent':'#818cf8','--accent-bg':'rgba(129,140,248,.08)','--accent-glow':'rgba(129,140,248,.4)',
      '--del-hover':'#f87171','--sync-fresh':'#34d399'
    },
    Smoke: {
      '--bg':'#1a1917','--bg-side':'#1e1d1b','--bg-hover':'#222120','--bg-card':'#1f1e1c','--bg-pop':'#252422',
      '--bd':'#2c2a28','--bd2':'#363430','--bd3':'#434040',
      '--tx':'#d4cfc9','--tx2':'#7a7570','--tx3':'#524e4a','--tx4':'#3a3735',
      '--tx-done':'#383532','--tx-head':'#eae5df',
      '--accent':'#a78bfa','--accent-bg':'rgba(167,139,250,.08)','--accent-glow':'rgba(167,139,250,.4)',
      '--del-hover':'#f87171','--sync-fresh':'#34d399'
    },
    Midnight: {
      '--bg':'#0a0c10','--bg-side':'#0d0f14','--bg-hover':'#111520','--bg-card':'#0f1219','--bg-pop':'#141820',
      '--bd':'#1a1e28','--bd2':'#222838','--bd3':'#2c3448',
      '--tx':'#b8c4d8','--tx2':'#5a6880','--tx3':'#3a4458','--tx4':'#2a3248',
      '--tx-done':'#242c3c','--tx-head':'#d8e4f4',
      '--accent':'#60a5fa','--accent-bg':'rgba(96,165,250,.07)','--accent-glow':'rgba(96,165,250,.4)',
      '--del-hover':'#f87171','--sync-fresh':'#34d399'
    },
    Nord: {
      '--bg':'#2e3440','--bg-side':'#272c37','--bg-hover':'#363c4a','--bg-card':'#2e3440','--bg-pop':'#3b4252',
      '--bd':'#3b4252','--bd2':'#434c5e','--bd3':'#4c566a',
      '--tx':'#d8dee9','--tx2':'#8a9ab0','--tx3':'#606a78','--tx4':'#434c5e',
      '--tx-done':'#3b4252','--tx-head':'#eceff4',
      '--accent':'#88c0d0','--accent-bg':'rgba(136,192,208,.08)','--accent-glow':'rgba(136,192,208,.4)',
      '--del-hover':'#bf616a','--sync-fresh':'#a3be8c'
    },
    Catppuccin: {
      '--bg':'#1e1e2e','--bg-side':'#181825','--bg-hover':'#27273f','--bg-card':'#1e1e2e','--bg-pop':'#313244',
      '--bd':'#313244','--bd2':'#45475a','--bd3':'#585b70',
      '--tx':'#cdd6f4','--tx2':'#a6adc8','--tx3':'#6c7086','--tx4':'#45475a',
      '--tx-done':'#313244','--tx-head':'#e6e9ef',
      '--accent':'#cba6f7','--accent-bg':'rgba(203,166,247,.07)','--accent-glow':'rgba(203,166,247,.4)',
      '--del-hover':'#f38ba8','--sync-fresh':'#a6e3a1'
    },
    Gruvbox: {
      '--bg':'#282828','--bg-side':'#1d2021','--bg-hover':'#32302f','--bg-card':'#252525','--bg-pop':'#3c3836',
      '--bd':'#3c3836','--bd2':'#504945','--bd3':'#665c54',
      '--tx':'#ebdbb2','--tx2':'#928374','--tx3':'#665c54','--tx4':'#504945',
      '--tx-done':'#3c3836','--tx-head':'#fbf1c7',
      '--accent':'#fe8019','--accent-bg':'rgba(254,128,25,.07)','--accent-glow':'rgba(254,128,25,.4)',
      '--del-hover':'#cc241d','--sync-fresh':'#b8bb26'
    },
    Dracula: {
      '--bg':'#282a36','--bg-side':'#21222c','--bg-hover':'#2d2f3f','--bg-card':'#282a36','--bg-pop':'#343646',
      '--bd':'#3d3f50','--bd2':'#44475a','--bd3':'#6272a4',
      '--tx':'#f8f8f2','--tx2':'#6272a4','--tx3':'#44475a','--tx4':'#3d3f50',
      '--tx-done':'#3d3f50','--tx-head':'#f8f8f2',
      '--accent':'#bd93f9','--accent-bg':'rgba(189,147,249,.07)','--accent-glow':'rgba(189,147,249,.4)',
      '--del-hover':'#ff5555','--sync-fresh':'#50fa7b'
    }
  }

  const LIGHT_THEMES = {
    Parchment: {
      '--bg':'#eceae6','--bg-side':'#e5e3de','--bg-hover':'#e0deda','--bg-card':'#f2f0ec','--bg-pop':'#f4f2ee',
      '--bd':'#d4d1cc','--bd2':'#c8c5bf','--bd3':'#b8b5af',
      '--tx':'#32302c','--tx2':'#706d68','--tx3':'#9e9b96','--tx4':'#b8b5b0',
      '--tx-done':'#c0bdb8','--tx-head':'#1e1c19',
      '--accent':'#5254c8','--accent-bg':'rgba(82,84,200,.07)','--accent-glow':'rgba(82,84,200,.25)',
      '--del-hover':'#cc4f4f','--sync-fresh':'#3a9e6a'
    },
    Paper: {
      '--bg':'#ffffff','--bg-side':'#f8f8f8','--bg-hover':'#f0f0f0','--bg-card':'#fafafa','--bg-pop':'#ffffff',
      '--bd':'#e0e0e0','--bd2':'#d0d0d0','--bd3':'#c0c0c0',
      '--tx':'#202020','--tx2':'#606060','--tx3':'#909090','--tx4':'#b0b0b0',
      '--tx-done':'#c8c8c8','--tx-head':'#101010',
      '--accent':'#4f46e5','--accent-bg':'rgba(79,70,229,.06)','--accent-glow':'rgba(79,70,229,.2)',
      '--del-hover':'#cc4f4f','--sync-fresh':'#3a9e6a'
    },
    Linen: {
      '--bg':'#f6f4f1','--bg-side':'#f0ede9','--bg-hover':'#ebe8e4','--bg-card':'#faf8f5','--bg-pop':'#fdfbf8',
      '--bd':'#ddd9d4','--bd2':'#d0ccc7','--bd3':'#c0bbb5',
      '--tx':'#2c2a27','--tx2':'#6a6660','--tx3':'#96928c','--tx4':'#b4b0aa',
      '--tx-done':'#c4c0ba','--tx-head':'#1a1816',
      '--accent':'#5254c8','--accent-bg':'rgba(82,84,200,.07)','--accent-glow':'rgba(82,84,200,.2)',
      '--del-hover':'#cc4f4f','--sync-fresh':'#3a9e6a'
    },
    Mist: {
      '--bg':'#f0f2f4','--bg-side':'#e8eaed','--bg-hover':'#e2e4e8','--bg-card':'#f5f6f8','--bg-pop':'#f8f9fb',
      '--bd':'#d4d8dc','--bd2':'#c8ccd0','--bd3':'#b8bcc0',
      '--tx':'#28303a','--tx2':'#606878','--tx3':'#909aa8','--tx4':'#b0b8c4',
      '--tx-done':'#c0c8d4','--tx-head':'#18222e',
      '--accent':'#3b82f6','--accent-bg':'rgba(59,130,246,.07)','--accent-glow':'rgba(59,130,246,.2)',
      '--del-hover':'#cc4f4f','--sync-fresh':'#3a9e6a'
    },
    Nordic: {
      '--bg':'#eceff4','--bg-side':'#e5e9f0','--bg-hover':'#d8dee9','--bg-card':'#f0f4f8','--bg-pop':'#f5f8ff',
      '--bd':'#d0d8e4','--bd2':'#b8c4d4','--bd3':'#9aaec4',
      '--tx':'#2e3440','--tx2':'#4c566a','--tx3':'#6a7a8e','--tx4':'#8c9ab0',
      '--tx-done':'#b0bccb','--tx-head':'#1e2530',
      '--accent':'#5e81ac','--accent-bg':'rgba(94,129,172,.07)','--accent-glow':'rgba(94,129,172,.2)',
      '--del-hover':'#bf616a','--sync-fresh':'#a3be8c'
    },
    Solarized: {
      '--bg':'#fdf6e3','--bg-side':'#eee8d5','--bg-hover':'#e8e2cf','--bg-card':'#fdf6e3','--bg-pop':'#f5efdc',
      '--bd':'#e0dac8','--bd2':'#cdc7b5','--bd3':'#b8b2a0',
      '--tx':'#586e75','--tx2':'#657b83','--tx3':'#839496','--tx4':'#93a1a1',
      '--tx-done':'#a8b4b8','--tx-head':'#002b36',
      '--accent':'#268bd2','--accent-bg':'rgba(38,139,210,.07)','--accent-glow':'rgba(38,139,210,.2)',
      '--del-hover':'#dc322f','--sync-fresh':'#859900'
    },
    Catppuccin: {
      '--bg':'#eff1f5','--bg-side':'#e6e9ef','--bg-hover':'#dce0e8','--bg-card':'#f2f4f8','--bg-pop':'#f8f9ff',
      '--bd':'#ccd0da','--bd2':'#bcc0cc','--bd3':'#acb0be',
      '--tx':'#4c4f69','--tx2':'#6c6f85','--tx3':'#8c8fa1','--tx4':'#acafc0',
      '--tx-done':'#bcc0cc','--tx-head':'#32354a',
      '--accent':'#8839ef','--accent-bg':'rgba(136,57,239,.06)','--accent-glow':'rgba(136,57,239,.2)',
      '--del-hover':'#d20f39','--sync-fresh':'#40a02b'
    },
    Rosé: {
      '--bg':'#fff5f7','--bg-side':'#fce8ed','--bg-hover':'#f9dce3','--bg-card':'#fff8fa','--bg-pop':'#fffcfd',
      '--bd':'#f0ccd6','--bd2':'#e8b8c4','--bd3':'#dda0b2',
      '--tx':'#3d1f2a','--tx2':'#8a5066','--tx3':'#b48090','--tx4':'#d0a0b0',
      '--tx-done':'#e0b8c8','--tx-head':'#2a1018',
      '--accent':'#db2777','--accent-bg':'rgba(219,39,119,.06)','--accent-glow':'rgba(219,39,119,.2)',
      '--del-hover':'#be123c','--sync-fresh':'#059669'
    }
  }

  /* ── theme state ──────────────────────────────────── */
  let theme        = localStorage.getItem('odak_theme')       || 'dark'
  let selectedDark = localStorage.getItem('odak_dark_theme')  || 'Catppuccin'
  let selectedLight= localStorage.getItem('odak_light_theme') || 'Nordic'

  function applyTheme(t, animate = true, slow = false) {
    const root = document.documentElement
    const cls = slow ? 'theme-transitioning-slow' : 'theme-transitioning'
    if (animate) root.classList.add(cls)
    requestAnimationFrame(() => {
      root.classList.toggle('light', t === 'light')
      const tokens = t === 'dark' ? DARK_THEMES[selectedDark] : LIGHT_THEMES[selectedLight]
      if (tokens) for (const [k, v] of Object.entries(tokens)) root.style.setProperty(k, v)
      if (animate) setTimeout(() => root.classList.remove(cls), slow ? 900 : 400)
    })
  }

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark'
    localStorage.setItem('odak_theme', theme)
    applyTheme(theme, true, true)
  }

  function pickTheme(name, mode) {
    if (mode === 'dark') {
      selectedDark = name
      localStorage.setItem('odak_dark_theme', name)
    } else {
      selectedLight = name
      localStorage.setItem('odak_light_theme', name)
    }
    const crossMode = theme !== mode
    if (crossMode) { theme = mode; localStorage.setItem('odak_theme', theme) }
    applyTheme(theme, true, crossMode)
  }

  /* ── theme popover ────────────────────────────────── */
  let showThemePicker = false
  let pickerLeaveTimer = null

  function pickerEnter() {
    if (pickerLeaveTimer) { clearTimeout(pickerLeaveTimer); pickerLeaveTimer = null }
    showThemePicker = true
  }

  function pickerLeave() {
    pickerLeaveTimer = setTimeout(() => { showThemePicker = false }, 220)
  }

  /* ── auth ─────────────────────────────────────────── */
  let inputUser = '', inputPass = ''
  let rememberMe = !!localStorage.getItem('odak_token')
  let authed = !!(localStorage.getItem('odak_token') || sessionStorage.getItem('odak_token'))
  let loginError = '', loginLoading = false

  /* ── state ────────────────────────────────────────── */
  let sections      = []
  let allItems      = []
  let activeSection = null    // null = All view
  let showDone      = false
  let addText       = ''
  let addTarget     = null    // section picker in All view
  let sectPickOpen  = false
  let addEl
  let movingId  = null
  let editingId = null, editText = ''
  let lastSync  = null

  /* ── feature 1: page title ────────────────────────── */
  $: if (typeof document !== 'undefined') {
    const open = allItems.filter(i => !i.done).length
    document.title = open > 0 ? `(${open}) odak` : 'odak'
  }

  /* ── feature 2: checkbox pop animation ───────────── */
  let poppingID = null

  /* ── keyboard item focus ──────────────────────────── */
  let focusedItemId = null

  $: flatItems = grouped.flatMap(g => g.items)

  function navFocus(dir) {
    const list = flatItems
    if (!list.length) return
    if (!focusedItemId) { focusedItemId = dir > 0 ? list[0].id : list[list.length - 1].id; scrollToFocused(); return }
    const idx = list.findIndex(i => i.id === focusedItemId)
    if (idx === -1) { focusedItemId = list[0].id; scrollToFocused(); return }
    const next = idx + dir
    if (next >= 0 && next < list.length) { focusedItemId = list[next].id; scrollToFocused() }
  }

  function scrollToFocused() {
    tick().then(() => {
      document.querySelector(`[data-item-id="${focusedItemId}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    })
  }

  /* ── feature 5: section collapse ─────────────────── */
  function loadCollapsed() {
    try {
      const raw = localStorage.getItem('odak-collapsed-sections')
      return raw ? new Set(JSON.parse(raw)) : new Set()
    } catch { return new Set() }
  }

  let collapsedSections = loadCollapsed()

  function toggleCollapse(name) {
    const s = new Set(collapsedSections)
    if (s.has(name)) s.delete(name)
    else s.add(name)
    collapsedSections = s
    localStorage.setItem('odak-collapsed-sections', JSON.stringify([...s]))
  }

  /* ── feature 4: keyboard shortcut overlay ─────────── */
  let showHelp = false

  const SHORTCUTS = [
    { key: 'j / ↓',    desc: 'move focus down' },
    { key: 'k / ↑',    desc: 'move focus up' },
    { key: 'x',        desc: 'toggle done on focused item' },
    { key: 'e',        desc: 'edit focused item text' },
    { key: 'd',        desc: 'delete focused item' },
    { key: 'n or /',   desc: 'focus add input' },
    { key: 'w',        desc: 'toggle #work filter' },
    { key: 'p',        desc: 'toggle #personal filter' },
    { key: 'r',        desc: 'refresh' },
    { key: '?',        desc: 'show this overlay' },
    { key: 'Esc',      desc: 'clear focus / close / cancel' },
    { key: 'Enter',    desc: 'confirm add / confirm edit' },
    { key: 'dblclick', desc: 'edit item text' },
  ]

  function closeHelp() { showHelp = false }

  /* ── feature 6: drag to reorder / move ───────────── */
  let dragId          = null
  let dragSection     = null
  let dragOverId      = null
  let dragOverPos     = null   // 'before' | 'after'
  let dragOverSection = null   // section name for cross-section drop target

  function dragCleanup() {
    dragId = dragSection = dragOverId = dragOverPos = dragOverSection = null
  }

  function onDragStart(e, item) {
    dragId      = item.id
    dragSection = item.section
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', item.id)
  }

  function onDragOver(e, item) {
    if (!dragId) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (item.section === dragSection) {
      dragOverSection = null
      const rect = e.currentTarget.getBoundingClientRect()
      dragOverId  = item.id
      dragOverPos = e.clientY < rect.top + rect.height / 2 ? 'before' : 'after'
    } else {
      dragOverId = dragOverPos = null
      dragOverSection = item.section
    }
  }

  function onDragLeave(e, item) {
    if (dragOverId === item.id) { dragOverId = null; dragOverPos = null }
    if (dragOverSection === item.section) dragOverSection = null
  }

  function onDragOverSection(e, name) {
    if (!dragId || name === dragSection) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    dragOverSection = name
    dragOverId = dragOverPos = null
  }

  function onDragLeaveSection(name) {
    if (dragOverSection === name) dragOverSection = null
  }

  async function onDrop(e, item) {
    e.preventDefault()
    if (!dragId) { dragCleanup(); return }
    if (item.section === dragSection) {
      if (dragId === item.id) { dragCleanup(); return }
      const secItems = allItems.filter(i => i.section === dragSection && !i.parent_id)
      const ids = secItems.map(i => i.id)
      const fromIdx = ids.indexOf(dragId)
      if (fromIdx === -1) { dragCleanup(); return }
      ids.splice(fromIdx, 1)
      const insertAt = dragOverPos === 'before' ? ids.indexOf(item.id) : ids.indexOf(item.id) + 1
      ids.splice(insertAt < 0 ? ids.length : insertAt, 0, dragId)
      dragCleanup()
      await api('POST', '/todos/reorder', { section: item.section, ids })
      poll()
    } else {
      const o = orig(dragId)
      if (o) { mutated(); o.section = item.section; allItems = allItems }
      dragCleanup()
      api('POST', `/todos/${o?.id ?? dragId}/move`, { section: item.section })
    }
  }

  async function onDropSection(e, name) {
    e.preventDefault()
    if (!dragId || name === dragSection) { dragCleanup(); return }
    const o = orig(dragId)
    if (o) { mutated(); o.section = name; allItems = allItems }
    dragCleanup()
    api('POST', `/todos/${o?.id ?? dragId}/move`, { section: name })
  }

  function onDragEnd() { dragCleanup() }

  /* ── tree helpers ─────────────────────────────────── */
  function treeFlat(items, collapsed) {
    const map = {}
    items.forEach(i => { map[i.id] = { ...i, _ch: [] } })
    const roots = []
    items.forEach(i => {
      const node = map[i.id]
      if (i.parent_id && map[i.parent_id]) map[i.parent_id]._ch.push(node)
      else roots.push(node)
    })
    function walk(nodes, d) {
      return nodes.flatMap(n => {
        const hasKids = n._ch.length > 0
        const isCollapsed = hasKids && collapsed.has(n.id)
        return [
          { ...n, _d: d, _hasKids: hasKids, _collapsed: isCollapsed },
          ...(isCollapsed ? [] : walk(n._ch, d + 1))
        ]
      })
    }
    return walk(roots, 0)
  }

  /* ── sub-item collapse ────────────────────────────── */
  function loadExpanded() {
    try {
      const raw = localStorage.getItem('odak-expanded-items')
      return raw ? new Set(JSON.parse(raw)) : new Set()
    } catch { return new Set() }
  }
  let expandedParents = loadExpanded()
  $: parentIds = new Set(allItems.filter(i => i.parent_id).map(i => i.parent_id))
  $: collapsedParents = new Set([...parentIds].filter(id => !expandedParents.has(id)))

  function toggleChildCollapse(id) {
    const s = new Set(expandedParents)
    if (s.has(id)) s.delete(id)
    else s.add(id)
    expandedParents = s
    localStorage.setItem('odak-expanded-items', JSON.stringify([...s]))
  }

  /* ── tag filter ───────────────────────────────────── */
  let activeTags = []
  function toggleTag(tag) {
    activeTags = activeTags.includes(tag) ? activeTags.filter(t => t !== tag) : [...activeTags, tag]
  }

  /* ── derived ──────────────────────────────────────── */
  $: allTags = [...new Set(allItems.flatMap(i => i.tags || []))].sort()
  $: tagFiltered = activeTags.length === 0
    ? allItems
    : allItems.filter(i => (i.tags || []).some(t => activeTags.includes(t)))
  $: visible = showDone ? tagFiltered : tagFiltered.filter(i => !i.done)

  $: grouped = (activeSection === null
    ? sections
        .filter(s => allItems.some(i => i.section === s.name))
        .map(s => ({
          name: s.name,
          collapsed: collapsedSections.has(s.name),
          items: collapsedSections.has(s.name) ? [] : treeFlat(visible.filter(i => i.section === s.name), collapsedParents)
        }))
    : [{ name: activeSection, collapsed: false, items: treeFlat(visible.filter(i => i.section === activeSection), collapsedParents) }])

  $: totalOpen = allItems.filter(i => !i.done).length
  $: effectiveAdd = activeSection ?? addTarget ?? sections[0]?.name ?? 'Inbox'

  /* ── api ──────────────────────────────────────────── */
  function getToken() {
    return localStorage.getItem('odak_token') || sessionStorage.getItem('odak_token') || ''
  }
  function h() { return { Authorization: `Bearer ${getToken()}` } }

  async function api(method, path, body) {
    const opts = { method, headers: h() }
    if (body !== undefined) {
      opts.headers['Content-Type'] = 'application/json'
      opts.body = JSON.stringify(body)
    }
    try {
      const r = await fetch(path, opts)
      if (r.status === 401) { authed = false; return null }
      if (r.status === 204 || !r.ok) return null
      return r.json()
    } catch { return null }
  }

  /* ── data ─────────────────────────────────────────── */
  async function load() {
    const [s, it] = await Promise.all([api('GET', '/sections'), api('GET', '/todos')])
    sections = s || []; allItems = it || []; lastSync = Date.now()
  }

  async function poll() {
    if (!authed || editingId) return
    const [s, it] = await Promise.all([api('GET', '/sections'), api('GET', '/todos')])
    if (s) sections = s
    if (it) { allItems = it; lastSync = Date.now() }
  }

  /* ── websocket ────────────────────────────────────── */
  let ws = null
  let wsEnabled = false
  let wsReconnectTimer = null
  let pollTimer = null

  let _pollDebounce = null
  let _lastMutation = 0
  function mutated() { _lastMutation = Date.now() }
  function debouncedPoll() {
    if (Date.now() - _lastMutation < 600) return  // own mutation in flight — skip
    if (_pollDebounce) return
    _pollDebounce = setTimeout(() => { _pollDebounce = null; poll() }, 120)
  }

  function connectWS() {
    if (!wsEnabled || !getToken()) return
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${proto}//${location.host}/ws?token=${getToken()}`)
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        if (msg.type === 'reload') debouncedPoll()
      } catch { /* ignore non-JSON */ }
    }
    ws.onclose = () => {
      ws = null
      if (wsEnabled) wsReconnectTimer = setTimeout(connectWS, 2000)
    }
    ws.onerror = () => {
      ws?.close()
    }
  }

  function disconnectWS() {
    wsEnabled = false
    if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
    if (ws) { ws.close(); ws = null }
  }

  onMount(() => {
    applyTheme(theme, false)
    if (authed) {
      load().then(() => {
        wsEnabled = true
        connectWS()
      })
    }
    pollTimer = setInterval(poll, 60000)
  })
  onDestroy(() => {
    clearInterval(pollTimer)
    disconnectWS()
  })

  /* ── nav ──────────────────────────────────────────── */
  function setSection(name) {
    activeSection = name; movingId = editingId = null; sectPickOpen = false
    tick().then(() => addEl?.focus())
  }

  /* ── mutations ────────────────────────────────────── */
  function orig(id) { return allItems.find(i => i.id === id) }

  async function toggle(item) {
    const o = orig(item.id); if (!o) return
    mutated(); o.done = !o.done; allItems = allItems
    poppingID = item.id
    setTimeout(() => { poppingID = null }, 250)
    api('PATCH', `/todos/${item.id}/done`)
  }

  let deletingIds = new Set()

  async function del(item) {
    mutated()
    deletingIds = new Set([...deletingIds, item.id])
    api('DELETE', `/todos/${item.id}`)
    setTimeout(() => {
      deletingIds = new Set([...deletingIds].filter(id => id !== item.id))
      const desc = new Set()
      function collect(id) {
        allItems.filter(i => i.parent_id === id).forEach(i => { desc.add(i.id); collect(i.id) })
      }
      collect(item.id); desc.add(item.id)
      allItems = allItems.filter(i => !desc.has(i.id))
    }, 180)
  }

  async function move(item, target) {
    const o = orig(item.id)
    mutated(); movingId = null
    if (o) o.section = target
    allItems = allItems
    api('POST', `/todos/${item.id}/move`, { section: target })
  }

  /* ── edit ─────────────────────────────────────────── */
  async function startEdit(item) {
    editingId = item.id; editText = item.text
    await tick(); document.querySelector('[data-edit]')?.focus()
  }

  async function commitEdit(item) {
    if (!editingId) return
    const text = editText.trim(); editingId = null
    if (!text || text === item.text) return
    const o = orig(item.id); if (!o) return
    mutated(); o.text = text; allItems = allItems
    api('PATCH', `/todos/${item.id}`, { text })
  }

  /* ── add ──────────────────────────────────────────── */
  function parseAdd(raw) {
    let text = raw.trim(), urgent = false, tags = [], deadline = null
    if (/^!\s/.test(text)) { urgent = true; text = text.replace(/^!\s*/, '') }
    text = text.replace(/#(\S+)/g, (_, t) => { tags.push(t); return '' })
    text = text.replace(/\bd:(\S+)/g, (_, d) => { deadline = d; return '' })
    return { text: text.replace(/\s+/g, ' ').trim(), urgent,
             tags: tags.length ? tags : undefined, deadline: deadline || undefined }
  }

  async function add() {
    const raw = addText.trim(); if (!raw) return
    addText = ''
    mutated()
    const created = await api('POST', '/todos', { ...parseAdd(raw), section: effectiveAdd })
    if (created) allItems = [...allItems, created]
    addEl?.focus()
  }

  /* ── auth ─────────────────────────────────────────── */
  async function login() {
    if (!inputUser.trim() || !inputPass) return
    loginError = ''; loginLoading = true
    try {
      const r = await fetch('/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user: inputUser.trim(), password: inputPass })
      })
      const data = await r.json()
      if (!r.ok) { loginError = data.error || 'login failed'; return }
      rememberMe
        ? (localStorage.setItem('odak_token', data.token), sessionStorage.removeItem('odak_token'))
        : (sessionStorage.setItem('odak_token', data.token), localStorage.removeItem('odak_token'))
      authed = true
      load().then(() => {
        wsEnabled = true
        connectWS()
      })
    } catch { loginError = 'could not reach server' }
    finally  { loginLoading = false }
  }

  function logout() {
    disconnectWS()
    localStorage.removeItem('odak_token'); sessionStorage.removeItem('odak_token')
    authed = false; inputUser = ''; inputPass = ''
    allItems = []; sections = []; activeSection = null
  }

  /* ── helpers ──────────────────────────────────────── */
  const PALETTE = ['#6b7db8','#5a9970','#8a7d4a','#7a6a9a','#4a8799','#8a6a50','#5a7a8a','#8a6070','#6a8a6a']
  function tagColor(t) {
    let h = 0; for (const c of t) h = (h * 31 + c.charCodeAt(0)) >>> 0
    return PALETTE[h % PALETTE.length]
  }

  function dlInfo(d) {
    if (!d) return null
    try {
      const dt = new Date(d.includes('T') ? d : d + 'T00:00:00')
      const now = new Date(); now.setHours(0,0,0,0)
      const diff = Math.round((dt - now) / 86400000)
      if (diff < 0)   return { label: d,             cls: 'overdue' }
      if (diff === 0) return { label: 'today',        cls: 'today'   }
      if (diff === 1) return { label: 'tomorrow',     cls: 'soon'    }
      if (diff <= 7)  return { label: `in ${diff}d`,  cls: 'soon'    }
      return { label: d, cls: '' }
    } catch { return { label: d, cls: '' } }
  }

  function secCount(name) { return sections.find(s => s.name === name)?.count ?? 0 }
  const ICON = { Focus:'⚡',Today:'☀',Next:'›',Backlog:'·',Someday:'○',Recurring:'↻',Inbox:'✦',Done:'✓' }
  function secIcon(n) { return ICON[n] || '·' }

  function focus(node) { node.focus(); return {} }

  function outsideClick(e) {
    if (movingId     && !e.target.closest('[data-move]')) movingId = null
    if (sectPickOpen && !e.target.closest('[data-sp]'))   sectPickOpen = false
    if (editingId    && !e.target.closest('[data-edit]')) {
      const item = allItems.find(i => i.id === editingId)
      if (item) commitEdit(item)
    }
    if (showHelp && e.target.classList.contains('help-overlay')) closeHelp()
  }

  let _now = Date.now(); setInterval(() => _now = Date.now(), 5000)
  function elapsed(ts) {
    const s = Math.round((_now - ts) / 1000)
    if (s < 5)  return 'just now'
    if (s < 60) return `${s}s ago`
    return `${Math.round(s/60)}m ago`
  }
  $: syncLabel = lastSync ? elapsed(lastSync) : ''
</script>

<svelte:window
  on:click={outsideClick}
  on:keydown={e => {
    if (!authed) { if (e.key === 'Enter') login(); return }
    if (showHelp) { if (e.key === 'Escape') closeHelp(); return }
    if (editingId) { if (e.key === 'Escape') editingId = null; return }
    if (e.key === '?') { e.preventDefault(); showHelp = true; return }
    if (e.key === 'Escape') { focusedItemId = null; movingId = null; sectPickOpen = false; addEl?.blur(); return }

    const notInput = document.activeElement?.tagName !== 'INPUT'
    if (!notInput) return

    if (e.key === '/' || e.key === 'n') { e.preventDefault(); addEl?.focus(); return }
    if (e.key === 'r') { poll(); return }
    if (e.key === 'w') { toggleTag('work'); return }
    if (e.key === 'p') { toggleTag('personal'); return }

    if (e.key === 'j' || e.key === 'ArrowDown') { e.preventDefault(); navFocus(1); return }
    if (e.key === 'k' || e.key === 'ArrowUp')   { e.preventDefault(); navFocus(-1); return }

    if (focusedItemId) {
      const item = flatItems.find(i => i.id === focusedItemId)
      if (!item) return
      if (e.key === 'x') { toggle(item); return }
      if (e.key === 'e') { startEdit(item); return }
      if (e.key === 'd') { del(item); focusedItemId = null; return }
    }
  }}
/>

<style>
  /* ── theme tokens ─────────────────────────────────── */
  :global(html) {
    --bg:          #111111;
    --bg-side:     #141414;
    --bg-hover:    #181818;
    --bg-card:     #161616;
    --bg-pop:      #1c1c1c;
    --bd:          #222222;
    --bd2:         #2c2c2c;
    --bd3:         #383838;
    --tx:          #d0d0d0;
    --tx2:         #6a6a6a;
    --tx3:         #484848;
    --tx4:         #323232;
    --tx-done:     #303030;
    --tx-head:     #eaeaea;
    --accent:      #818cf8;
    --accent-bg:   rgba(129,140,248,.08);
    --accent-glow: rgba(129,140,248,.4);
    --del-hover:   #f87171;
    --sync-fresh:  #34d399;
  }
  :global(html.light) {
    --bg:          #f6f4f1;
    --bg-side:     #f0ede9;
    --bg-hover:    #ebe8e4;
    --bg-card:     #faf8f5;
    --bg-pop:      #fdfbf8;
    --bd:          #ddd9d4;
    --bd2:         #d0ccc7;
    --bd3:         #c0bbb5;
    --tx:          #2c2a27;
    --tx2:         #6a6660;
    --tx3:         #96928c;
    --tx4:         #b4b0aa;
    --tx-done:     #c4c0ba;
    --tx-head:     #1a1816;
    --accent:      #5254c8;
    --accent-bg:   rgba(82,84,200,.07);
    --accent-glow: rgba(82,84,200,.2);
    --del-hover:   #cc4f4f;
    --sync-fresh:  #3a9e6a;
  }

  :global(*) { box-sizing: border-box; margin: 0; padding: 0 }

  /* same-mode theme swap — quick */
  :global(html.theme-transitioning *) {
    transition:
      background-color 300ms ease,
      color 220ms ease,
      border-color 220ms ease,
      box-shadow 220ms ease !important;
  }
  /* cross-mode (dark↔light) — slower, calmer */
  :global(html.theme-transitioning-slow *) {
    transition:
      background-color 700ms ease-in-out,
      color 550ms ease-in-out,
      border-color 550ms ease-in-out,
      box-shadow 550ms ease-in-out !important;
  }
  :global(html, body) { height: 100% }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
    background: var(--bg);
    color: var(--tx);
    -webkit-font-smoothing: antialiased;
    font-size: 14px;
  }

  /* ── shell ────────────────────────────────────────── */
  .app { display: flex; height: 100vh; overflow: hidden }

  /* ── sidebar ──────────────────────────────────────── */
  .sidebar {
    width: 200px; flex-shrink: 0;
    display: flex; flex-direction: column;
    background: var(--bg-side);
    border-right: 1px solid var(--bd);
  }

  .brand {
    padding: 20px 16px 14px;
    display: flex; align-items: center; gap: 8px; flex-shrink: 0;
  }
  .brand-dot {
    width: 8px; height: 8px; border-radius: 50%;
    background: var(--accent); box-shadow: 0 0 8px var(--accent-glow); flex-shrink: 0;
  }
  .brand-name {
    font-size: 12px; font-weight: 600; color: var(--tx3);
    letter-spacing: .14em; text-transform: uppercase;
  }

  .nav { flex: 1; overflow-y: auto; padding: 0 8px }
  .nav::-webkit-scrollbar { display: none }

  .nav-label {
    font-size: 10px; font-weight: 600; color: var(--tx4);
    letter-spacing: .1em; text-transform: uppercase; padding: 8px 10px 4px;
  }

  .nav-btn {
    display: flex; align-items: center; gap: 8px;
    width: 100%; padding: 6px 10px;
    border: none; background: none; color: var(--tx2);
    font-family: inherit; font-size: 13px;
    text-align: left; cursor: pointer; border-radius: 6px;
    transition: background .1s, color .1s;
    white-space: nowrap; overflow: hidden;
  }
  .nav-btn:hover  { background: var(--bg-hover); color: var(--tx) }
  .nav-btn.active { background: color-mix(in srgb, var(--accent) 10%, var(--bg-hover)); color: var(--tx-head) }
  .nav-btn.nav-all { font-weight: 500; margin-bottom: 4px }

  .nav-icon { font-size: 11px; width: 16px; text-align: center; flex-shrink: 0; opacity: .45 }
  .nav-btn.active .nav-icon { opacity: 1; color: var(--accent) }

  .nav-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis }
  .nav-count {
    font-size: 11px; font-variant-numeric: tabular-nums;
    color: var(--tx4); flex-shrink: 0; min-width: 14px; text-align: right;
    transition: color .1s;
  }
  .nav-btn.active .nav-count { color: var(--tx3) }

  /* ── section collapse ─────────────────────────────── */
  .nav-item-wrap {
    display: flex; align-items: center; gap: 1px;
  }
  .nav-item-wrap .nav-btn { flex: 1; min-width: 0 }

  .nav-collapse-btn {
    flex-shrink: 0; background: none; border: none;
    padding: 4px 5px; border-radius: 4px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    opacity: 0; transition: opacity .1s, background .1s;
  }
  .nav-item-wrap:hover .nav-collapse-btn { opacity: 1 }
  .nav-collapse-btn:hover { background: var(--bg-hover) }

  .nav-arrow {
    font-size: 9px; color: var(--tx4);
  }

  .divider { height: 1px; background: var(--bd); margin: 6px 10px }

  .sidebar-foot {
    padding: 10px 16px 14px; border-top: 1px solid var(--bd); flex-shrink: 0;
  }
  .sync-row { display: flex; align-items: center; gap: 5px; margin-bottom: 8px }
  .sync-dot {
    width: 5px; height: 5px; border-radius: 50%;
    background: var(--bd3); flex-shrink: 0; transition: background .4s;
  }
  .sync-dot.fresh { background: var(--sync-fresh) }
  .sync-txt { font-size: 10px; color: var(--tx4) }

  .logout-btn {
    width: 100%; padding: 5px 8px;
    background: none; border: 1px solid var(--bd); border-radius: 5px;
    color: var(--tx4); font-family: inherit; font-size: 11px;
    cursor: pointer; text-align: center; transition: color .1s, border-color .1s;
  }
  .logout-btn:hover { color: var(--tx2); border-color: var(--bd2) }

  /* ── main ─────────────────────────────────────────── */
  .main { flex: 1; display: flex; flex-direction: column; overflow: hidden; min-width: 0 }

  .toolbar {
    padding: 18px 24px 0;
    display: flex; align-items: center; gap: 10px; flex-shrink: 0;
  }
  .toolbar-title { font-size: 16px; font-weight: 600; color: var(--tx-head); flex: 1 }

  .tb-btn {
    background: none; border: 1px solid var(--bd); border-radius: 5px;
    color: var(--tx4); font-family: inherit; font-size: 11px;
    padding: 4px 9px; cursor: pointer; transition: color .1s, border-color .1s;
    white-space: nowrap; line-height: 1.4;
  }
  .tb-btn:hover { color: var(--tx2); border-color: var(--bd2) }
  .tb-btn.on { color: var(--accent); border-color: var(--accent); background: var(--accent-bg) }

  /* ── help button ──────────────────────────────────── */
  .help-btn {
    background: none; border: 1px solid var(--bd); border-radius: 5px;
    color: var(--tx4); font-family: inherit; font-size: 11px;
    padding: 4px 8px; cursor: pointer; transition: color .1s, border-color .1s;
    line-height: 1.4; flex-shrink: 0;
  }
  .help-btn:hover { color: var(--tx2); border-color: var(--bd2) }

  /* ── keyboard shortcut overlay ────────────────────── */
  .help-overlay {
    position: fixed; inset: 0;
    background: rgba(0,0,0,.55); backdrop-filter: blur(2px);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000;
  }
  .help-card {
    background: var(--bg-pop); border: 1px solid var(--bd3); border-radius: 12px;
    padding: 24px 28px; min-width: 340px; max-width: 480px;
    box-shadow: 0 24px 60px rgba(0,0,0,.4);
  }
  .help-title {
    font-size: 13px; font-weight: 600; color: var(--tx-head);
    margin-bottom: 16px; letter-spacing: .04em;
  }
  .help-table { width: 100%; border-collapse: collapse }
  .help-table tr + tr td { padding-top: 7px }
  .help-key {
    font-family: monospace; font-size: 12px;
    background: var(--bg-hover); border: 1px solid var(--bd3); border-radius: 4px;
    padding: 2px 7px; color: var(--accent); white-space: nowrap;
    width: 1%; padding-right: 12px;
  }
  .help-desc { font-size: 12px; color: var(--tx2); padding-left: 4px }
  .help-close {
    margin-top: 18px; width: 100%; background: none;
    border: 1px solid var(--bd2); border-radius: 6px;
    color: var(--tx3); font-family: inherit; font-size: 12px;
    padding: 6px; cursor: pointer; transition: color .1s, border-color .1s;
  }
  .help-close:hover { color: var(--tx); border-color: var(--bd3) }

  /* ── theme picker wrapper ─────────────────────────── */
  .theme-wrap {
    position: relative; flex-shrink: 0;
  }

  .theme-btn {
    background: none; border: none;
    color: var(--tx4); padding: 4px 6px;
    cursor: pointer; line-height: 0; border-radius: 5px;
    transition: color .15s;
  }
  .theme-btn:hover { color: var(--tx2) }

  /* ── theme popover ────────────────────────────────── */
  .theme-pop {
    position: absolute; right: 0; top: calc(100% + 8px);
    background: var(--bg-pop); border: 1px solid var(--bd3); border-radius: 10px;
    padding: 8px; z-index: 400;
    box-shadow: 0 16px 40px rgba(0,0,0,.3);
    display: flex; gap: 6px;
    pointer-events: auto;
  }

  .theme-col {
    width: 150px; display: flex; flex-direction: column; gap: 2px;
  }

  .theme-col-hdr {
    font-size: 10px; font-weight: 600; color: var(--tx4);
    letter-spacing: .1em; text-transform: uppercase;
    padding: 2px 7px 5px;
  }

  .theme-row {
    display: flex; align-items: center; gap: 7px;
    width: 100%; background: none; border: none;
    color: var(--tx2); font-family: inherit; font-size: 12px;
    padding: 5px 7px; border-radius: 6px; cursor: pointer;
    text-align: left; transition: background .1s, color .1s;
    white-space: nowrap;
  }
  .theme-row:hover { background: var(--bg-hover); color: var(--tx) }
  .theme-row.sel   { color: var(--tx-head) }

  .theme-swatch {
    width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
    border: 1px solid rgba(128,128,128,.2);
  }

  .theme-check {
    margin-left: auto; font-size: 10px; color: var(--accent); flex-shrink: 0;
  }

  .theme-sep {
    width: 1px; background: var(--bd2); margin: 4px 0; flex-shrink: 0;
  }

  /* ── tag strip ────────────────────────────────────── */
  .tag-strip {
    display: flex; align-items: center; gap: 4px;
    padding: 10px 24px 0; flex-shrink: 0;
    overflow-x: auto; scrollbar-width: none;
  }
  .tag-strip::-webkit-scrollbar { display: none }
  .tag-chip {
    background: none; border: 1px solid var(--bd2); border-radius: 20px;
    color: var(--tc); font-family: inherit; font-size: 11px; font-weight: 500;
    padding: 2px 9px; cursor: pointer; white-space: nowrap; flex-shrink: 0;
    transition: background .1s, border-color .1s, opacity .15s;
    opacity: .55;
  }
  .tag-chip:hover  { opacity: .9; border-color: var(--tc) }
  .tag-chip.active {
    background: color-mix(in srgb, var(--tc) 14%, transparent);
    border-color: var(--tc); opacity: 1;
  }
  .tag-chip-clear {
    background: none; border: 1px solid var(--bd); border-radius: 20px;
    color: var(--tx3); font-family: inherit; font-size: 13px; line-height: 1;
    padding: 1px 8px; cursor: pointer; flex-shrink: 0;
    transition: color .1s, border-color .1s;
  }
  .tag-chip-clear:hover { color: var(--tx); border-color: var(--bd2) }

  /* ── add bar ──────────────────────────────────────── */
  .add-wrap { padding: 12px 24px 8px; flex-shrink: 0 }

  .add-box {
    display: flex; align-items: center; gap: 8px;
    background: var(--bg-card); border: 1px solid var(--bd2); border-radius: 8px;
    padding: 2px 12px; transition: border-color .15s, box-shadow .15s;
  }
  .add-box:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-bg) }

  .add-plus { font-size: 16px; color: var(--tx4); flex-shrink: 0; transition: color .15s; line-height: 1 }
  .add-box:focus-within .add-plus { color: var(--accent) }

  .add-input {
    flex: 1; background: none; border: none; outline: none;
    color: var(--tx); font-family: inherit; font-size: 13px;
    padding: 9px 0; caret-color: var(--accent);
  }
  .add-input::placeholder { color: var(--tx4) }

  /* section picker */
  .sect-pick { position: relative; flex-shrink: 0 }
  .sect-pick-btn {
    background: var(--bg-hover); border: 1px solid var(--bd2); border-radius: 5px;
    color: var(--tx3); font-family: inherit; font-size: 11px;
    padding: 3px 8px; cursor: pointer; white-space: nowrap;
    transition: color .1s, border-color .1s;
  }
  .sect-pick-btn:hover { color: var(--tx); border-color: var(--bd3) }

  .sect-pick-drop {
    position: absolute; right: 0; top: calc(100% + 5px);
    background: var(--bg-pop); border: 1px solid var(--bd3); border-radius: 8px;
    padding: 4px; min-width: 120px; z-index: 300;
    box-shadow: 0 12px 32px rgba(0,0,0,.25);
  }
  .sect-pick-opt {
    display: flex; align-items: center; gap: 7px; width: 100%;
    background: none; border: none; color: var(--tx2); font-family: inherit; font-size: 12px;
    padding: 5px 8px; border-radius: 5px; cursor: pointer;
    text-align: left; transition: background .1s, color .1s; white-space: nowrap;
  }
  .sect-pick-opt:hover { background: var(--bg-hover); color: var(--tx) }
  .sect-pick-opt.active { color: var(--accent) }

  /* ── scroll area ──────────────────────────────────── */
  .scroll { flex: 1; overflow-y: auto; padding: 4px 24px 32px }
  .scroll::-webkit-scrollbar { width: 5px }
  .scroll::-webkit-scrollbar-track { background: transparent }
  .scroll::-webkit-scrollbar-thumb { background: var(--bd2); border-radius: 10px }

  /* ── section group ────────────────────────────────── */
  .sec-group { margin-bottom: 24px }
  .sec-group:last-child { margin-bottom: 0 }

  .sec-hdr {
    display: flex; align-items: center; gap: 8px;
    padding: 6px 10px; margin: 0 -10px 2px;
    position: sticky; top: 0; z-index: 10;
    background: var(--bg);
    border-bottom: 1px solid var(--bd);
  }
  .sec-hdr-icon { font-size: 11px; opacity: .45; flex-shrink: 0 }
  .sec-hdr-name {
    font-size: 11px; font-weight: 600; flex: 1;
    color: var(--tx3); letter-spacing: .06em; text-transform: uppercase;
  }
  .sec-hdr-count { font-size: 10px; color: var(--tx4); font-variant-numeric: tabular-nums }
  .sec-hdr-arrow { font-size: 8px; color: var(--tx4); margin-left: 4px }
  .sec-hdr { cursor: pointer }
  .sec-hdr:hover .sec-hdr-name { color: var(--tx2) }
  .sec-hdr.drag-over-section {
    background: var(--accent-bg);
    border-bottom-color: var(--accent);
    color: var(--accent);
  }
  .sec-hdr.drag-over-section .sec-hdr-name { color: var(--accent) }

  /* ── item row ─────────────────────────────────────── */
  .item {
    display: flex; align-items: center; gap: 8px;
    padding: 5px 10px; margin: 0 -10px;
    border-radius: 6px; transition: background .1s; position: relative;
  }
  .item:hover { background: var(--bg-hover) }

  /* ── delete animation ────────────────────────────── */
  .item.deleting {
    animation: itemDelete 180ms ease forwards;
    pointer-events: none;
  }
  @keyframes itemDelete {
    to { opacity: 0; transform: translateX(-28px); }
  }

  /* ── keyboard focus ──────────────────────────────── */
  .item.focused {
    background: var(--accent-bg);
    box-shadow: inset 2px 0 0 var(--accent);
  }

  /* ── drag and drop ────────────────────────────────── */
  .item.drag-over-before { border-top: 2px solid var(--accent) }
  .item.drag-over-after  { border-bottom: 2px solid var(--accent) }

  .drag-handle {
    font-size: 12px; color: var(--tx4); flex-shrink: 0;
    opacity: 0; cursor: grab; transition: opacity .1s;
    user-select: none; margin-right: -4px;
  }
  .item:hover .drag-handle { opacity: .5 }
  .drag-handle:hover { opacity: 1 !important }

  .urgent-rule {
    position: absolute; left: 0; top: 8px; bottom: 8px;
    width: 2px; border-radius: 1px;
    background: rgba(248,113,113,.5);
  }

  .tree-lines {
    display: flex; align-items: flex-start; flex-shrink: 0; align-self: stretch;
  }
  .tree-seg {
    width: 18px; height: 100%; flex-shrink: 0; position: relative;
  }
  .tree-seg.vert::before {
    content: ''; position: absolute;
    left: 8px; top: -5px; bottom: -5px; width: 1px;
    background: var(--bd2);
  }
  .tree-seg.leaf::before {
    content: ''; position: absolute;
    left: 8px; top: -5px; bottom: 50%; width: 1px;
    background: var(--bd2);
  }
  .tree-seg.leaf::after {
    content: ''; position: absolute;
    left: 8px; top: 50%; width: 9px; height: 1px;
    background: var(--bd2);
  }

  /* ── sub-item collapse toggle ─────────────────────── */
  .collapse-toggle {
    width: 12px; height: 12px; flex-shrink: 0;
    display: flex; align-items: center; justify-content: center;
    background: none; border: none; padding: 0; margin-right: 1px;
    color: var(--tx4); font-size: 7px; cursor: default;
    opacity: 0; transition: opacity .1s, color .1s;
  }
  .collapse-toggle.has-kids { cursor: pointer }
  .item:hover .collapse-toggle.has-kids { opacity: .55 }
  .collapse-toggle.has-kids.is-collapsed { opacity: .8; color: var(--tx3) }

  /* ── checkbox ─────────────────────────────────────── */
  .cb {
    width: 14px; height: 14px; flex-shrink: 0;
    border: 1.5px solid var(--bd3); border-radius: 4px;
    display: flex; align-items: center; justify-content: center;
    cursor: pointer; transition: border-color .15s, background .15s;
  }
  .cb:hover { border-color: var(--accent) }
  .cb.done  { background: var(--accent); border-color: var(--accent) }
  .cb.done::after {
    content: ''; display: block; width: 6px; height: 4px;
    border-left: 1.5px solid rgba(255,255,255,.9);
    border-bottom: 1.5px solid rgba(255,255,255,.9);
    transform: translateY(-1px) rotate(-45deg);
  }

  /* feature 2: checkbox pop animation */
  .cb-pop { animation: cbPop 200ms ease }
  @keyframes cbPop {
    0%   { transform: scale(1) }
    50%  { transform: scale(1.35) }
    100% { transform: scale(1) }
  }

  .content { flex: 1; min-width: 0; display: flex; align-items: baseline; flex-wrap: wrap; gap: 5px }

  .item-text {
    font-size: 13px; color: var(--tx); line-height: 1.5;
    cursor: text; word-break: break-word;
  }
  .item.is-done .item-text { color: var(--tx-done); text-decoration: line-through }

  .edit-input {
    flex: 1 1 100%; min-width: 0;
    background: var(--bg-card); border: 1px solid var(--accent); border-radius: 4px;
    color: var(--tx); font-family: inherit; font-size: 13px;
    padding: 2px 7px; outline: none; caret-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-bg);
  }

  .tag-il { font-size: 11px; font-weight: 500; white-space: nowrap }
  .dl-il  { font-size: 11px; white-space: nowrap }
  .dl-il.overdue { color: #f87171 }
  .dl-il.today   { color: #f0b429 }
  .dl-il.soon    { color: #60a5fa }
  .dl-il         { color: var(--tx3) }

  .actions {
    display: flex; align-items: center; gap: 1px;
    opacity: 0; transition: opacity .1s; flex-shrink: 0; margin-top: 1px;
  }
  .item:hover .actions { opacity: 1 }

  .act {
    background: none; border: none; color: var(--tx4);
    cursor: pointer; font-size: 11px; padding: 3px 6px; border-radius: 4px;
    font-family: inherit; transition: color .1s, background .1s;
    white-space: nowrap; line-height: 1;
  }
  .act:hover     { background: var(--bg-hover); color: var(--tx2) }
  .act.del:hover { color: var(--del-hover) }

  .move-wrap { position: relative }
  .popover {
    position: absolute; right: 0; top: calc(100% + 4px);
    background: var(--bg-pop); border: 1px solid var(--bd3); border-radius: 8px;
    padding: 4px; min-width: 120px; z-index: 200;
    box-shadow: 0 12px 32px rgba(0,0,0,.25);
  }
  .pop-opt {
    display: flex; align-items: center; gap: 7px; width: 100%;
    background: none; border: none; color: var(--tx2);
    font-family: inherit; font-size: 12px; padding: 5px 8px; border-radius: 5px;
    cursor: pointer; text-align: left; transition: background .1s, color .1s; white-space: nowrap;
  }
  .pop-opt:hover { background: var(--bg-hover); color: var(--tx) }
  .pop-icon { font-size: 10px; width: 13px; text-align: center; opacity: .45 }

  .empty { display: flex; flex-direction: column; align-items: center; padding: 48px 0; color: var(--tx4); gap: 8px }
  .empty-glyph { font-size: 22px }
  .empty-text  { font-size: 13px }

  .skeleton { display: flex; flex-direction: column; gap: 2px; padding: 4px 0 }
  .skel-row  { display: flex; align-items: center; gap: 8px; padding: 7px 10px }
  .skel-cb   { width: 14px; height: 14px; border-radius: 4px; background: var(--bg-hover); flex-shrink: 0 }
  .skel-line { height: 12px; border-radius: 4px; background: var(--bg-hover); animation: pulse 1.6s ease-in-out infinite }
  @keyframes pulse { 0%,100%{opacity:.35} 50%{opacity:.65} }

  /* ── login ────────────────────────────────────────── */
  .login-wrap { display: flex; align-items: center; justify-content: center; height: 100vh; background: var(--bg) }
  .login-card {
    width: 340px; background: var(--bg-card); border: 1px solid var(--bd2);
    border-radius: 14px; padding: 36px 32px; display: flex; flex-direction: column; gap: 14px;
  }
  .login-logo { display: flex; align-items: center; gap: 10px; margin-bottom: 4px }
  .login-dot  { width: 10px; height: 10px; border-radius: 50%; background: var(--accent); box-shadow: 0 0 10px var(--accent-glow) }
  .login-name { font-size: 20px; font-weight: 600; color: var(--tx-head) }
  .login-sub  { font-size: 12px; color: var(--tx3); line-height: 1.5 }
  .login-field { display: flex; flex-direction: column; gap: 6px }
  .login-label { font-size: 11px; color: var(--tx3); font-weight: 500 }
  .login-input {
    background: var(--bg); border: 1px solid var(--bd2); border-radius: 8px;
    color: var(--tx); font-family: inherit; font-size: 14px; padding: 10px 14px;
    outline: none; width: 100%; transition: border-color .15s, box-shadow .15s;
  }
  .login-input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-bg) }
  .login-input.err  { border-color: rgba(248,113,113,.5) }

  .remember-row { display: flex; align-items: center; gap: 8px; cursor: pointer; user-select: none }
  .remember-cb  {
    width: 14px; height: 14px; border: 1.5px solid var(--bd3); border-radius: 4px;
    display: flex; align-items: center; justify-content: center; flex-shrink: 0;
    transition: border-color .15s, background .15s;
  }
  .remember-cb.on { background: var(--accent); border-color: var(--accent) }
  .remember-cb.on::after {
    content: ''; display: block; width: 6px; height: 4px;
    border-left: 1.5px solid rgba(255,255,255,.9);
    border-bottom: 1.5px solid rgba(255,255,255,.9);
    transform: translateY(-1px) rotate(-45deg);
  }
  .remember-label { font-size: 12px; color: var(--tx2) }

  .login-btn {
    background: var(--accent); border: none; border-radius: 8px;
    color: #fff; font-family: inherit; font-size: 14px; font-weight: 500;
    padding: 10px; cursor: pointer; width: 100%; margin-top: 2px;
    transition: opacity .15s, transform .1s;
  }
  .login-btn:hover   { opacity: .88 }
  .login-btn:active  { transform: scale(.99) }
  .login-btn:disabled { opacity: .55; cursor: default }
  .login-err { font-size: 12px; color: #f87171; text-align: center }

</style>

<!-- ── Login ──────────────────────────────────────── -->
{#if !authed}
  <div class="login-wrap">
    <div class="login-card" in:fly={{ y: 10, duration: 260, easing: cubicOut }}>
      <div class="login-logo">
        <div class="login-dot"></div>
        <div class="login-name">odak</div>
      </div>
      <div class="login-sub">todos, backed by markdown</div>

      <div class="login-field">
        <label class="login-label" for="lu">Username</label>
        <input id="lu" class="login-input" class:err={!!loginError}
          type="text" autocomplete="username" placeholder="username"
          bind:value={inputUser}
          on:keydown={e => e.key === 'Enter' && document.getElementById('lp').focus()}
          autofocus />
      </div>
      <div class="login-field">
        <label class="login-label" for="lp">Password</label>
        <input id="lp" class="login-input" class:err={!!loginError}
          type="password" autocomplete="current-password" placeholder="password"
          bind:value={inputPass}
          on:keydown={e => e.key === 'Enter' && login()} />
      </div>

      <label class="remember-row">
        <div class="remember-cb" class:on={rememberMe} on:click={() => rememberMe = !rememberMe}></div>
        <span class="remember-label">Remember me</span>
      </label>

      {#if loginError}
        <div class="login-err" in:fade={{ duration: 150 }}>{loginError}</div>
      {/if}
      <button class="login-btn" on:click={login} disabled={loginLoading}>
        {loginLoading ? 'connecting…' : 'sign in →'}
      </button>
    </div>
  </div>

<!-- ── App ───────────────────────────────────────── -->
{:else}
  <!-- feature 4: keyboard shortcut overlay -->
  {#if showHelp}
    <div class="help-overlay" in:fade={{ duration: 120 }} out:fade={{ duration: 100 }}>
      <div class="help-card" in:scale={{ duration: 140, start: .95, easing: cubicOut }}>
        <div class="help-title">Keyboard shortcuts</div>
        <table class="help-table">
          {#each SHORTCUTS as s}
            <tr>
              <td class="help-key">{s.key}</td>
              <td class="help-desc">{s.desc}</td>
            </tr>
          {/each}
        </table>
        <button class="help-close" on:click={closeHelp}>close</button>
      </div>
    </div>
  {/if}

  <div class="app">

    <!-- Sidebar -->
    <nav class="sidebar">
      <div class="brand">
        <div class="brand-dot"></div>
        <div class="brand-name">odak</div>
      </div>

      <div class="nav">
        <button class="nav-btn nav-all" class:active={activeSection === null} on:click={() => setSection(null)}>
          <span class="nav-icon">≡</span>
          <span class="nav-name">All items</span>
          <span class="nav-count">{totalOpen > 0 ? totalOpen : ''}</span>
        </button>

        <div class="divider"></div>
        <div class="nav-label">sections</div>

        <!-- feature 5: collapsible sections in sidebar -->
        {#each sections as s (s.name)}
          <div class="nav-item-wrap">
            <button class="nav-btn" class:active={activeSection === s.name}
              on:click={() => setSection(s.name)}>
              <span class="nav-icon">{secIcon(s.name)}</span>
              <span class="nav-name">{s.name}</span>
              <span class="nav-count">{s.count > 0 ? s.count : ''}</span>
            </button>
            <button class="nav-collapse-btn" title="{collapsedSections.has(s.name) ? 'expand' : 'collapse'}"
              on:click|stopPropagation={() => toggleCollapse(s.name)}>
              <span class="nav-arrow">{collapsedSections.has(s.name) ? '▶' : '▼'}</span>
            </button>
          </div>
        {/each}
      </div>

      <div class="sidebar-foot">
        <div class="sync-row">
          <span class="sync-dot" class:fresh={lastSync && (_now - lastSync < 30000)}></span>
          <span class="sync-txt">{syncLabel || 'syncing…'}</span>
        </div>
        <button class="logout-btn" on:click={logout}>sign out</button>
      </div>
    </nav>

    <!-- Main -->
    <div class="main">

      <!-- Toolbar -->
      <div class="toolbar">
        <div class="toolbar-title">{activeSection ?? 'All items'}</div>
        <button class="tb-btn" class:on={showDone} on:click={() => showDone = !showDone}>
          {showDone ? 'hide done' : 'show done'}
        </button>

        <!-- feature 4: help button -->
        <button class="help-btn" on:click={() => showHelp = true} title="keyboard shortcuts (?)">?</button>

        <!-- Theme picker wrapper -->
        <div class="theme-wrap"
          on:mouseenter={pickerEnter}
          on:mouseleave={pickerLeave}
        >
          <button class="theme-btn" on:click={toggleTheme} title="toggle theme">
            {#if theme === 'dark'}
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round">
                <circle cx="7" cy="7" r="2.5"/>
                <line x1="7" y1="0.5" x2="7" y2="2.2"/>
                <line x1="7" y1="11.8" x2="7" y2="13.5"/>
                <line x1="0.5" y1="7" x2="2.2" y2="7"/>
                <line x1="11.8" y1="7" x2="13.5" y2="7"/>
                <line x1="2.55" y1="2.55" x2="3.75" y2="3.75"/>
                <line x1="10.25" y1="10.25" x2="11.45" y2="11.45"/>
                <line x1="11.45" y1="2.55" x2="10.25" y2="3.75"/>
                <line x1="3.75" y1="10.25" x2="2.55" y2="11.45"/>
              </svg>
            {:else}
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">
                <path d="M10.5 8.8A4.3 4.3 0 0 1 5.2 3.5 4.3 4.3 0 1 0 10.5 8.8z"/>
              </svg>
            {/if}
          </button>

          {#if showThemePicker}
            <div class="theme-pop"
              in:scale={{ duration: 120, start: .94, easing: cubicOut }}
              on:mouseenter={pickerEnter}
              on:mouseleave={pickerLeave}
            >
              <!-- Dark column -->
              <div class="theme-col">
                <div class="theme-col-hdr">Dark</div>
                {#each Object.entries(DARK_THEMES) as [name, tokens]}
                  <button class="theme-row"
                    class:sel={selectedDark === name}
                    on:click={() => pickTheme(name, 'dark')}
                  >
                    <span class="theme-swatch" style="background:{tokens['--bg']}"></span>
                    {name}
                    {#if selectedDark === name}
                      <span class="theme-check">✓</span>
                    {/if}
                  </button>
                {/each}
              </div>

              <div class="theme-sep"></div>

              <!-- Light column -->
              <div class="theme-col">
                <div class="theme-col-hdr">Light</div>
                {#each Object.entries(LIGHT_THEMES) as [name, tokens]}
                  <button class="theme-row"
                    class:sel={selectedLight === name}
                    on:click={() => pickTheme(name, 'light')}
                  >
                    <span class="theme-swatch" style="background:{tokens['--bg']}"></span>
                    {name}
                    {#if selectedLight === name}
                      <span class="theme-check">✓</span>
                    {/if}
                  </button>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Tag filter strip -->
      {#if allTags.length > 0}
        <div class="tag-strip">
          {#each allTags as tag}
            <button class="tag-chip" class:active={activeTags.includes(tag)}
              style="--tc:{tagColor(tag)}"
              on:click={() => toggleTag(tag)}>#{tag}</button>
          {/each}
          {#if activeTags.length > 0}
            <button class="tag-chip-clear" on:click={() => activeTags = []}>×</button>
          {/if}
        </div>
      {/if}

      <!-- Add bar -->
      <div class="add-wrap">
        <div class="add-box">
          <span class="add-plus">+</span>
          <input class="add-input"
            placeholder="add a task… (! urgent  #tag  d:date)"
            bind:value={addText} bind:this={addEl}
            on:keydown={e => e.key === 'Enter' && add()} />
          {#if activeSection === null}
            <div class="sect-pick" data-sp>
              <button class="sect-pick-btn" data-sp on:click={() => sectPickOpen = !sectPickOpen}>
                {effectiveAdd} ▾
              </button>
              {#if sectPickOpen}
                <div class="sect-pick-drop" data-sp
                  in:scale={{ duration: 110, start: .94, easing: cubicOut }}>
                  {#each sections as s}
                    <button class="sect-pick-opt"
                      class:active={effectiveAdd === s.name}
                      data-sp on:click={() => { addTarget = s.name; sectPickOpen = false }}>
                      <span style="opacity:.4;font-size:10px">{secIcon(s.name)}</span>{s.name}
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>

      <!-- Items -->
      <div class="scroll">
        {#if !lastSync && allItems.length === 0}
          <div class="skeleton">
            {#each [55,38,68,42] as w}
              <div class="skel-row">
                <div class="skel-cb"></div>
                <div class="skel-line" style="width:{w}%"></div>
              </div>
            {/each}
          </div>

        {:else if grouped.length === 0}
          <div class="empty">
            <div class="empty-glyph">○</div>
            <div class="empty-text">{showDone ? 'nothing here' : 'all done'}</div>
          </div>

        {:else}
          {#each grouped as group (group.name)}
            <div class="sec-group">
              {#if activeSection === null}
                <div class="sec-hdr"
                  class:drag-over-section={dragOverSection === group.name}
                  on:click={() => toggleCollapse(group.name)}
                  on:dragover={e => onDragOverSection(e, group.name)}
                  on:dragleave={() => onDragLeaveSection(group.name)}
                  on:drop={e => onDropSection(e, group.name)}
                >
                  <span class="sec-hdr-icon">{secIcon(group.name)}</span>
                  <span class="sec-hdr-name">{group.name}</span>
                  <span class="sec-hdr-count">{group.collapsed ? '…' : group.items.length || ''}</span>
                  <span class="sec-hdr-arrow">{group.collapsed ? '▶' : '▼'}</span>
                </div>
              {/if}

              {#each group.items as item (item.id)}
                {@const dl = dlInfo(item.deadline)}
                {@const d  = item._d ?? 0}

                <!-- feature 3: out:fly for delete animation; feature 6: draggable -->
                <div class="item"
                  class:is-done={item.done}
                  class:urgent={item.urgent}
                  class:focused={focusedItemId === item.id}
                  class:drag-over-before={dragOverId === item.id && dragOverPos === 'before'}
                  class:drag-over-after={dragOverId === item.id && dragOverPos === 'after'}
                  class:deleting={deletingIds.has(item.id)}
                  data-item-id={item.id}
                  on:click={() => focusedItemId = item.id}
                  on:dragover={e => onDragOver(e, item)}
                  on:dragleave={e => onDragLeave(e, item)}
                  on:drop={e => onDrop(e, item)}
                >
                  {#if item.urgent}<div class="urgent-rule"></div>{/if}

                  <!-- feature 6: drag handle -->
                  <span class="drag-handle" title="drag to reorder"
                    draggable="true"
                    on:dragstart={e => onDragStart(e, item)}
                    on:dragend={onDragEnd}
                  >⠿</span>

                  <!-- Tree connector lines -->
                  {#if d > 0}
                    <div class="tree-lines">
                      {#each Array(d) as _, i}
                        <div class="tree-seg" class:vert={i < d - 1} class:leaf={i === d - 1}></div>
                      {/each}
                    </div>
                  {/if}

                  <!-- Sub-item collapse toggle -->
                  <button class="collapse-toggle"
                    class:has-kids={item._hasKids}
                    class:is-collapsed={item._collapsed}
                    tabindex="-1"
                    on:click|stopPropagation={() => item._hasKids && toggleChildCollapse(item.id)}>
                    {#if item._hasKids}{item._collapsed ? '▶' : '▼'}{/if}
                  </button>

                  <!-- Checkbox — feature 2: cb-pop animation -->
                  <div class="cb" class:done={item.done} class:cb-pop={poppingID === item.id}
                    role="checkbox" aria-checked={item.done} tabindex="0"
                    on:click={() => toggle(item)}
                    on:keydown={e => (e.key===' '||e.key==='Enter') && toggle(item)}
                  ></div>

                  <!-- Text + inline meta -->
                  <div class="content">
                    {#if editingId === item.id}
                      <input class="edit-input" bind:value={editText} data-edit use:focus
                        on:keydown={e => { if (e.key==='Enter') commitEdit(item); if (e.key==='Escape') editingId=null }}
                        on:blur={() => commitEdit(item)} />
                    {:else}
                      <span class="item-text" on:dblclick={() => startEdit(item)}>{item.text}</span>
                      {#each (item.tags || []) as tag}
                        <span class="tag-il" style="color:{tagColor(tag)}">#<!--
                        -->{tag}</span>
                      {/each}
                      {#if dl}
                        <span class="dl-il {dl.cls || ''}">◷ {dl.label}</span>
                      {/if}
                      {#if item.trigger}
                        <span class="dl-il" style="opacity:.4">w:{item.trigger}</span>
                      {/if}
                    {/if}
                  </div>

                  <!-- Row actions -->
                  <div class="actions">
                    <div class="move-wrap" data-move>
                      <button class="act" data-move
                        on:click={() => movingId = movingId === item.id ? null : item.id}>move</button>
                      {#if movingId === item.id}
                        <div class="popover" data-move
                          in:scale={{ duration: 110, start: .94, easing: cubicOut }}>
                          {#each sections.filter(s => s.name !== item.section) as s}
                            <button class="pop-opt" data-move on:click={() => move(item, s.name)}>
                              <span class="pop-icon">{secIcon(s.name)}</span>{s.name}
                            </button>
                          {/each}
                        </div>
                      {/if}
                    </div>
                    <button class="act del" on:click={() => del(item)}>×</button>
                  </div>
                </div>
              {/each}
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}
