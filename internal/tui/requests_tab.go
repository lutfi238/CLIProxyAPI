package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// requestsViewMode controls whether the tab shows the file list or one file's body.
type requestsViewMode int

const (
	reqViewList requestsViewMode = iota
	reqViewDetail
)

// requestsTabModel renders a list of recent per-request log files plus an
// inline detail view for a selected file. It mirrors the visual style of
// usage_tab.go (cards/headers/colors) so the tab feels native.
type requestsTabModel struct {
	client *Client

	mode        requestsViewMode
	entries     []RequestLogSummary
	visible     []int // indices into entries after filtering/search
	cursor      int   // index into visible
	scroll      int   // top-of-list offset for keyboard scrolling
	loadEnabled bool  // whether RequestLog (full body capture) is on
	loadedAt    time.Time
	loadErr     error
	loading     bool

	// list filters
	errorsOnly bool
	searching  bool
	searchIn   textinput.Model
	searchTerm string

	// detail
	detailViewport viewport.Model
	detailReady    bool
	detailEntry    *RequestLogSummary
	detailContent  string
	detailErr      error
	detailLoading  bool

	width  int
	height int
	ready  bool
}

// requestsListMsg delivers fetched list entries to the model.
type requestsListMsg struct {
	entries []RequestLogSummary
	enabled bool
	err     error
}

// requestsDetailMsg delivers fetched file content to the model.
type requestsDetailMsg struct {
	id      string
	content []byte
	err     error
}

func newRequestsTabModel(client *Client) requestsTabModel {
	ti := textinput.New()
	ti.Prompt = T("reqlogs_search_prompt")
	ti.CharLimit = 128
	return requestsTabModel{
		client:   client,
		mode:     reqViewList,
		searchIn: ti,
	}
}

func (m requestsTabModel) Init() tea.Cmd {
	return m.fetchList
}

func (m requestsTabModel) fetchList() tea.Msg {
	entries, enabled, err := m.client.ListRequestLogs(0)
	return requestsListMsg{entries: entries, enabled: enabled, err: err}
}

func (m requestsTabModel) fetchDetail(id string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetRequestLogContent(id)
		return requestsDetailMsg{id: id, content: data, err: err}
	}
}

func (m *requestsTabModel) recomputeVisible() {
	m.visible = m.visible[:0]
	term := strings.ToLower(strings.TrimSpace(m.searchTerm))
	for i := range m.entries {
		entry := &m.entries[i]
		if m.errorsOnly && !entry.IsError {
			continue
		}
		if term != "" {
			haystack := strings.ToLower(entry.Name + " " + entry.Model + " " + entry.Provider + " " + entry.Effort + " " + entry.ID + " " + entry.Method)
			if !strings.Contains(haystack, term) {
				continue
			}
		}
		m.visible = append(m.visible, i)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(0, len(m.visible)-1)
	}
	if m.scroll > m.cursor {
		m.scroll = m.cursor
	}
}

func (m requestsTabModel) Update(msg tea.Msg) (requestsTabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case localeChangedMsg:
		m.searchIn.Prompt = T("reqlogs_search_prompt")
		if m.detailReady {
			m.detailViewport.SetContent(m.detailContent)
		}
		return m, nil

	case requestsListMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err
			m.loadEnabled = msg.enabled
			return m, nil
		}
		m.loadErr = nil
		m.entries = msg.entries
		m.loadEnabled = msg.enabled
		m.loadedAt = time.Now()
		m.recomputeVisible()
		return m, nil

	case requestsDetailMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.detailErr = msg.err
			m.detailContent = ""
		} else {
			m.detailErr = nil
			m.detailContent = string(msg.content)
		}
		if m.detailReady {
			m.detailViewport.SetContent(m.renderDetail())
			m.detailViewport.GotoTop()
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == reqViewDetail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}

	if m.mode == reqViewDetail && m.detailReady {
		var cmd tea.Cmd
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m requestsTabModel) updateList(msg tea.KeyMsg) (requestsTabModel, tea.Cmd) {
	if m.searching {
		switch msg.String() {
		case "enter":
			m.searchTerm = strings.TrimSpace(m.searchIn.Value())
			m.searching = false
			m.searchIn.Blur()
			m.recomputeVisible()
			return m, nil
		case "esc":
			m.searching = false
			m.searchIn.SetValue("")
			m.searchTerm = ""
			m.searchIn.Blur()
			m.recomputeVisible()
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchIn, cmd = m.searchIn.Update(msg)
			m.searchTerm = strings.TrimSpace(m.searchIn.Value())
			m.recomputeVisible()
			return m, cmd
		}
	}

	switch msg.String() {
	case "r":
		m.loading = true
		return m, m.fetchList
	case "f":
		m.errorsOnly = !m.errorsOnly
		m.recomputeVisible()
		return m, nil
	case "/":
		m.searching = true
		m.searchIn.Focus()
		return m, textinput.Blink
	case "esc":
		if m.searchTerm != "" {
			m.searchTerm = ""
			m.searchIn.SetValue("")
			m.recomputeVisible()
		}
		return m, nil
	case "j", "down":
		if len(m.visible) > 0 {
			m.cursor = (m.cursor + 1) % len(m.visible)
			m.adjustScroll()
		}
		return m, nil
	case "k", "up":
		if len(m.visible) > 0 {
			m.cursor = (m.cursor - 1 + len(m.visible)) % len(m.visible)
			m.adjustScroll()
		}
		return m, nil
	case "g":
		m.cursor = 0
		m.scroll = 0
		return m, nil
	case "G":
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
			m.adjustScroll()
		}
		return m, nil
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.visible) {
			entry := m.entries[m.visible[m.cursor]]
			m.detailEntry = &entry
			m.detailContent = ""
			m.detailErr = nil
			m.detailLoading = true
			m.mode = reqViewDetail
			if !m.detailReady {
				m.detailViewport = viewport.New(m.width, m.contentHeight()-1)
				m.detailReady = true
			} else {
				m.detailViewport.Width = m.width
				m.detailViewport.Height = m.contentHeight() - 1
			}
			m.detailViewport.SetContent(m.renderDetail())
			return m, m.fetchDetail(entry.ID)
		}
	}
	return m, nil
}

func (m requestsTabModel) updateDetail(msg tea.KeyMsg) (requestsTabModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = reqViewList
		return m, nil
	case "r":
		if m.detailEntry != nil {
			m.detailLoading = true
			m.detailErr = nil
			m.detailContent = ""
			m.detailViewport.SetContent(m.renderDetail())
			return m, m.fetchDetail(m.detailEntry.ID)
		}
		return m, nil
	}
	if m.detailReady {
		var cmd tea.Cmd
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *requestsTabModel) adjustScroll() {
	visibleRows := m.listVisibleRows()
	if visibleRows <= 0 {
		return
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visibleRows {
		m.scroll = m.cursor - visibleRows + 1
	}
}

func (m requestsTabModel) listVisibleRows() int {
	// header(2) + table header(2) + footer(1) ≈ 5 lines reserved.
	rows := m.contentHeight() - 5
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m requestsTabModel) contentHeight() int {
	if m.height > 0 {
		return m.height
	}
	return 20
}

func (m *requestsTabModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.searchIn.Width = w - 16
	if m.detailReady {
		m.detailViewport.Width = w
		m.detailViewport.Height = m.contentHeight() - 1
	}
	m.ready = true
}

func (m requestsTabModel) View() string {
	if !m.ready {
		return T("loading")
	}
	if m.mode == reqViewDetail {
		return m.renderDetailView()
	}
	return m.renderListView()
}

// ─────────── List view ───────────

func (m requestsTabModel) renderListView() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(T("reqlogs_title")))
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render(T("reqlogs_help_list")))
	sb.WriteString("\n")

	// Status / filter line
	statusBits := []string{}
	if m.loading {
		statusBits = append(statusBits, subtitleStyle.Render(T("reqlogs_loading")))
	} else if m.loadErr != nil {
		statusBits = append(statusBits, errorStyle.Render(T("error_prefix")+m.loadErr.Error()))
	} else {
		statusBits = append(statusBits, lipgloss.NewStyle().Foreground(colorMuted).Render(
			fmt.Sprintf("%d %s", len(m.visible), T("reqlogs_count"))))
		filterLabel := T("reqlogs_filter_all")
		if m.errorsOnly {
			filterLabel = warningStyle.Render(T("reqlogs_filter_errors"))
		}
		statusBits = append(statusBits, lipgloss.NewStyle().Foreground(colorMuted).Render("· "+T("logs_filter")+": ")+filterLabel)
		if !m.loadedAt.IsZero() {
			statusBits = append(statusBits, lipgloss.NewStyle().Foreground(colorMuted).Render(
				"· "+m.loadedAt.Local().Format("15:04:05")))
		}
	}
	if !m.loadEnabled {
		statusBits = append(statusBits, warningStyle.Render(T("reqlogs_disabled_hint")))
	}
	sb.WriteString(strings.Join(statusBits, "  "))
	sb.WriteString("\n")

	if m.searching {
		sb.WriteString(m.searchIn.View())
		sb.WriteString("\n")
	} else if m.searchTerm != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render(
			fmt.Sprintf("/%s", m.searchTerm)))
		sb.WriteString("\n")
	}

	if len(m.entries) == 0 {
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render(T("reqlogs_no_data")))
		return sb.String()
	}
	if len(m.visible) == 0 {
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render(T("reqlogs_no_match")))
		return sb.String()
	}

	// Table header
	sb.WriteString(strings.Repeat("─", minInt(m.width, 100)))
	sb.WriteString("\n")
	header := fmt.Sprintf("  %-8s %-6s %-10s %-22s %-8s %-6s %-7s",
		T("reqlogs_col_time"),
		T("reqlogs_col_method"),
		T("reqlogs_col_provider"),
		T("reqlogs_col_model"),
		T("reqlogs_col_effort"),
		T("reqlogs_col_status"),
		T("reqlogs_col_size"),
	)
	sb.WriteString(tableHeaderStyle.Render(header))
	sb.WriteString("\n")

	// Rows (with simple keyboard scrolling)
	rowsToShow := m.listVisibleRows()
	end := m.scroll + rowsToShow
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := m.scroll; i < end; i++ {
		entry := &m.entries[m.visible[i]]
		isCursor := i == m.cursor
		sb.WriteString(m.renderRow(entry, isCursor))
		sb.WriteString("\n")
	}

	// Position hint
	if len(m.visible) > rowsToShow {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render(
			fmt.Sprintf("  %d / %d", m.cursor+1, len(m.visible))))
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderRow renders one entry row.
func (m requestsTabModel) renderRow(entry *RequestLogSummary, selected bool) string {
	timeStr := "—"
	if entry.Modified > 0 {
		timeStr = time.Unix(entry.Modified, 0).Local().Format("15:04:05")
	}
	method := entry.Method
	if method == "" {
		method = "—"
	}
	provider := entry.Provider
	if provider == "" {
		provider = "—"
	}
	model := entry.Model
	if model == "" {
		model = "—"
	}
	effort := entry.Effort
	if effort == "" {
		effort = T("reqlogs_effort_none")
	}
	status := "—"
	if entry.Status > 0 {
		status = fmt.Sprintf("%d", entry.Status)
	}
	size := formatBytes(entry.Size)

	row := fmt.Sprintf("  %-8s %-6s %-10s %-22s %-8s %-6s %-7s",
		timeStr,
		truncate(method, 6),
		truncate(provider, 10),
		truncate(model, 22),
		truncate(effort, 8),
		truncate(status, 6),
		truncate(size, 7),
	)

	style := lipgloss.NewStyle()
	switch {
	case selected:
		style = tableSelectedStyle
	case entry.IsError:
		style = lipgloss.NewStyle().Foreground(colorError)
	case effortIsHigh(entry.Effort):
		style = lipgloss.NewStyle().Foreground(colorHighlight)
	default:
		style = tableCellStyle
	}
	return style.Render(row)
}

// effortIsHigh returns true for effort levels we want to surface visually.
func effortIsHigh(effort string) bool {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "high", "xhigh", "extra_high":
		return true
	}
	return false
}

// formatBytes renders a byte count in a compact human-friendly form.
func formatBytes(n int64) string {
	switch {
	case n <= 0:
		return "0"
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.1fG", float64(n)/(1024*1024*1024))
}

// ─────────── Detail view ───────────

func (m requestsTabModel) renderDetailView() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(T("reqlogs_title")))
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render(T("reqlogs_help_detail")))
	sb.WriteString("\n")

	if m.detailReady {
		sb.WriteString(m.detailViewport.View())
	} else {
		sb.WriteString(subtitleStyle.Render(T("reqlogs_loading_file")))
	}
	return sb.String()
}

// renderDetail returns the content shown inside the detail viewport.
func (m requestsTabModel) renderDetail() string {
	var sb strings.Builder
	if m.detailEntry != nil {
		sb.WriteString(m.renderDetailHeader(m.detailEntry))
		sb.WriteString("\n")
	}
	if m.detailLoading {
		sb.WriteString(subtitleStyle.Render(T("reqlogs_loading_file")))
		return sb.String()
	}
	if m.detailErr != nil {
		sb.WriteString(errorStyle.Render(T("error_prefix") + m.detailErr.Error()))
		return sb.String()
	}
	sb.WriteString(m.detailContent)
	return sb.String()
}

// renderDetailHeader prints a compact summary card for the selected request.
func (m requestsTabModel) renderDetailHeader(entry *RequestLogSummary) string {
	parts := []string{}
	add := func(label, value string) {
		if value == "" {
			return
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(colorMuted).Render(label+":")+" "+
			lipgloss.NewStyle().Foreground(colorText).Render(value))
	}
	timeStr := "—"
	if entry.Modified > 0 {
		timeStr = time.Unix(entry.Modified, 0).Local().Format("2006-01-02 15:04:05")
	}
	add(T("reqlogs_col_time"), timeStr)
	add(T("reqlogs_col_method"), entry.Method)
	add(T("reqlogs_col_provider"), entry.Provider)
	add(T("reqlogs_col_model"), entry.Model)
	if entry.Effort != "" {
		effortLabel := entry.Effort
		if effortIsHigh(entry.Effort) {
			effortLabel = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true).Render(entry.Effort)
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(colorMuted).Render(T("reqlogs_col_effort")+":")+" "+effortLabel)
	}
	if entry.Status > 0 {
		statusStr := fmt.Sprintf("%d", entry.Status)
		statusLabel := statusStr
		if entry.Status >= 400 {
			statusLabel = errorStyle.Render(statusStr)
		} else if entry.Status >= 300 {
			statusLabel = warningStyle.Render(statusStr)
		} else {
			statusLabel = successStyle.Render(statusStr)
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(colorMuted).Render(T("reqlogs_col_status")+":")+" "+statusLabel)
	}
	add(T("reqlogs_col_size"), formatBytes(entry.Size))
	add(T("reqlogs_col_id"), entry.ID)
	if entry.IsError {
		parts = append(parts, errorStyle.Render("● error log"))
	}
	return strings.Join(parts, "  ")
}

// sortByModifiedDesc keeps the list newest-first regardless of server order.
// Currently the server returns newest-first already; this helper exists so a
// future reordering on the server doesn't surprise the UI.
func sortByModifiedDesc(in []RequestLogSummary) []RequestLogSummary {
	out := make([]RequestLogSummary, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	return out
}

var _ = sortByModifiedDesc // exported reordering helper, currently unused
