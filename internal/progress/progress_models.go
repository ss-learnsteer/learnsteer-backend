package progress

// ProgressPageDTO represents the complete Progress & Achievements Dashboard payload
type ProgressPageDTO struct {
	CurrentZScore     CurrentZScoreDTO       `json:"current_z_score"`
	IslandRank        ProgressRankDTO        `json:"island_rank"`
	OverallCompletion OverallCompletionDTO   `json:"overall_completion"`
	ZScoreTrends      ZScoreTrendsDTO        `json:"z_score_trends"`
	WeeklyActivity    WeeklyActivityDTO      `json:"weekly_activity"`
	CurriculumMastery []CurriculumMasteryDTO `json:"curriculum_mastery"`
	RecentReports     []ReportItemDTO        `json:"recent_reports"`
	Achievements      AchievementsDTO        `json:"achievements"`
	DailyMotivation   DailyMotivationDTO     `json:"daily_motivation"`
	ReportDownloadURL string                 `json:"report_download_url"`
}

// CurrentZScoreDTO represents the top Z-score card
type CurrentZScoreDTO struct {
	Value          float64 `json:"value"`             // e.g. 1.842
	FormattedValue string  `json:"formatted_value"`   // e.g. "1.842"
	Increment      float64 `json:"increment"`         // e.g. +0.122
	IncrementLabel string  `json:"increment_label"`   // e.g. "+0.122"
	ProgressBarPct int     `json:"progress_bar_pct"`  // e.g. 72
}

// ProgressRankDTO represents the Island Rank card
type ProgressRankDTO struct {
	Rank          string `json:"rank"`           // e.g. "#24"
	RankNumber    int    `json:"rank_number"`    // e.g. 24
	Percentile    string `json:"percentile"`     // e.g. "Top 0.1%"
	NextMilestone string `json:"next_milestone"` // e.g. "Top 10 in District"
}

// OverallCompletionDTO represents the Overall Completion card
type OverallCompletionDTO struct {
	Percentage     int `json:"percentage"`      // e.g. 68
	CompletedUnits int `json:"completed_units"` // e.g. 3
	TotalUnits     int `json:"total_units"`     // e.g. 4
}

// TrendPointDTO represents a point on the Z-Score trend chart
type TrendPointDTO struct {
	Label       string  `json:"label"`       // e.g. "JAN", "2024"
	Value       float64 `json:"value"`       // e.g. 1.84
	Height      string  `json:"height"`      // e.g. "85%"
	Highlighted bool    `json:"highlighted"` // true for current
}

// ZScoreTrendsDTO holds both 6-month and yearly trends
type ZScoreTrendsDTO struct {
	SixMonths []TrendPointDTO `json:"six_months"`
	Yearly    []TrendPointDTO `json:"yearly"`
}

// ActivityBlockDTO represents an activity box on the heatmap
type ActivityBlockDTO struct {
	Day   string  `json:"day"`   // e.g. "Mon"
	Hrs   float64 `json:"hrs"`   // e.g. 1.2
	Level int     `json:"level"` // 0 to 4
}

// WeeklyActivityDTO represents the 4-week activity heatmap
type WeeklyActivityDTO struct {
	TotalHours float64              `json:"total_hours"` // e.g. 24.5
	TotalLabel string               `json:"total_label"` // e.g. "24.5 hrs total"
	Weeks      [][]ActivityBlockDTO `json:"weeks"`       // 4 weeks x 7 days
}

// CurriculumMasteryDTO represents per-subject mastery level
type CurriculumMasteryDTO struct {
	Subject    string `json:"subject"`     // e.g. "Biology"
	Percentage int    `json:"percentage"`  // e.g. 72
	Color      string `json:"color"`       // e.g. "bg-emerald-500"
	BarColor   string `json:"bar_color"`   // e.g. "#10B981"
}

// ReportItemDTO represents a recent performance report
type ReportItemDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`      // e.g. "June Monthly Analysis"
	Date       string `json:"date"`       // e.g. "Generated June 30, 2024"
	ColorClass string `json:"color_class"` // e.g. "bg-rose-50 text-rose-500"
	DownloadURL string `json:"download_url"`
}

// BadgeItemDTO represents an achievement badge
type BadgeItemDTO struct {
	ID          string `json:"id"`          // e.g. "bio-whiz"
	Name        string `json:"name"`        // e.g. "Bio Whiz"
	Unlocked    bool   `json:"unlocked"`    // true/false
	ColorClass  string `json:"color_class"` // e.g. "text-emerald-500 bg-emerald-50 border-emerald-500"
	Description string `json:"description"` // e.g. "Scored above 75% in 3 consecutive biology papers."
}

// AchievementsDTO holds unlocked badges
type AchievementsDTO struct {
	UnlockedCount int            `json:"unlocked_count"` // e.g. 3
	Badges        []BadgeItemDTO `json:"badges"`
}

// DailyMotivationDTO contains the daily quote
type DailyMotivationDTO struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}
