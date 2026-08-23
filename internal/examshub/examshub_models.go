package examshub

import "time"

// ExamsHubResponseDTO is the full payload for the Exams Hub page
type ExamsHubResponseDTO struct {
	LatestExam       LatestExamDTO        `json:"latest_exam"`
	UnitPerformance  []UnitPerformanceDTO `json:"unit_performance"`
	PredictedZScore  ZScoreDTO            `json:"predicted_z_score"`
	IslandRank       IslandRankDTO        `json:"island_rank"`
	Badge            BadgeDTO             `json:"badge"`
	AssessmentHistory AssessmentHistoryDTO `json:"assessment_history"`
	ReportURL        string               `json:"report_url"`
}

// LatestExamDTO contains the student's most recent exam results breakdown
type LatestExamDTO struct {
	ExamID           uint      `json:"exam_id"`
	ExamTitle        string    `json:"exam_title"`
	FinalScore       int       `json:"final_score"`       // Total score achieved
	MaxScore         int       `json:"max_score"`         // Maximum possible score (100)
	MCQMarks         int       `json:"mcq_marks"`         // MCQ marks achieved
	MCQMaxMarks      int       `json:"mcq_max_marks"`     // MCQ max (40)
	EssayMarks       int       `json:"essay_marks"`       // Structured Essay marks achieved
	EssayMaxMarks    int       `json:"essay_max_marks"`   // Essay max (60)
	MCQPercentage    int       `json:"mcq_percentage"`    // MCQ accuracy %
	EssayPercentage  int       `json:"essay_percentage"`  // Essay accuracy %
	OverallPercentage int      `json:"overall_percentage"`
	CompletedAt      *time.Time `json:"completed_at"`
	TimeTakenMin     int       `json:"time_taken_min"`
}

// UnitPerformanceDTO represents per-unit performance breakdown
type UnitPerformanceDTO struct {
	UnitName   string `json:"unit_name"`
	Percentage int    `json:"percentage"`
	Correct    int    `json:"correct"`
	Total      int    `json:"total"`
	BarColor   string `json:"bar_color"`
}

// ZScoreDTO represents the predicted Z-score and its recent change
type ZScoreDTO struct {
	Score     float64 `json:"score"`      // e.g. 1.82
	Increment float64 `json:"increment"`  // e.g. +0.14 from last exam
	Trend     string  `json:"trend"`      // "up", "down", "stable"
}

// IslandRankDTO represents the student's island-wide ranking and change
type IslandRankDTO struct {
	CurrentRank    int    `json:"current_rank"`     // e.g. 24
	PreviousRank   int    `json:"previous_rank"`    // e.g. 28
	PlacesChanged  int    `json:"places_changed"`   // e.g. 4 (positive = improved)
	Direction      string `json:"direction"`        // "up", "down", "stable"
	Message        string `json:"message"`          // e.g. "Up from #28 in the last mock exam."
}

// BadgeDTO represents any newly unlocked or most recent badge
type BadgeDTO struct {
	IsNew       bool   `json:"is_new"`        // true if unlocked in this session
	Name        string `json:"name"`          // e.g. "Bio Whiz"
	Description string `json:"description"`   // e.g. "Achieved 90%+ in 5 consecutive Bio assessments."
	Icon        string `json:"icon"`          // e.g. "🏆", "🔬", "⚡"
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
}

// AssessmentHistoryDTO shows recent performance trends
type AssessmentHistoryDTO struct {
	GrowthPct      int    `json:"growth_pct"`       // e.g. 4  (meaning +4%)
	GrowthLabel    string `json:"growth_label"`     // e.g. "+4% Growth"
	LastWeekPct    int    `json:"last_week_pct"`    // e.g. 82
	CurrentWeekPct int    `json:"current_week_pct"` // e.g. 86
	LastWeekLabel  string `json:"last_week_label"`  // "Last Week"
	CurrentWeekLabel string `json:"current_week_label"` // "Current Week"
	Trend          string `json:"trend"`            // "up", "down", "stable"
}
