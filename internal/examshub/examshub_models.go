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

// ===== Mock Exams Page DTOs =====

// MockExamsPageDTO is the full payload for the Mock Exams page
type MockExamsPageDTO struct {
	LastResult          LastResultDTO           `json:"last_result"`
	UpcomingSchedule    []UpcomingMockDTO       `json:"upcoming_schedule"`
	Eligibility         []EligibilityDTO        `json:"eligibility"`
	Strategy            StrategyDTO             `json:"strategy"`
}

// LastResultDTO shows the student's most recent mock exam result summary
type LastResultDTO struct {
	MockNumber   int    `json:"mock_number"`    // e.g. 3
	Label        string `json:"label"`          // e.g. "Mock #3: 72%"
	Percentage   int    `json:"percentage"`     // e.g. 72
	ExamTitle    string `json:"exam_title"`     // e.g. "Biology Mock #3"
	ExamID       uint   `json:"exam_id"`
}

// UpcomingMockDTO represents a scheduled mock exam in the calendar
type UpcomingMockDTO struct {
	ID            uint       `json:"id"`
	Month         string     `json:"month"`          // e.g. "MAY"
	Day           int        `json:"day"`            // e.g. 20
	Subject       string     `json:"subject"`        // e.g. "BIOLOGY"
	SubjectColor  string     `json:"subject_color"`  // e.g. "emerald", "violet", "orange"
	SessionLabel  string     `json:"session_label"`  // e.g. "Session #4"
	Title         string     `json:"title"`          // e.g. "Bio Mock #4 - May 20, 2025"
	TimeSlot      string     `json:"time_slot"`      // e.g. "08:30 AM - 11:30 AM"
	ExamType      string     `json:"exam_type"`      // e.g. "MCQ & Essay", "Full Syllabus"
	Action        string     `json:"action"`         // e.g. "Book a Seat", "Register Now", "Locked"
	IsLocked      bool       `json:"is_locked"`
	LockReason    string     `json:"lock_reason,omitempty"` // e.g. "Unlocks at 80% Progress"
	Date          time.Time  `json:"date"`
}

// EligibilityDTO shows subject revision progress for mock exam eligibility
type EligibilityDTO struct {
	Subject      string `json:"subject"`        // e.g. "BIOLOGY REVISION"
	Percentage   int    `json:"percentage"`     // e.g. 75
	IsUnlocked   bool   `json:"is_unlocked"`
	Message      string `json:"message"`        // e.g. "Unlocked for Mock #4" or "Need 70% for Mock #3"
	ActionURL    string `json:"action_url,omitempty"` // e.g. "/revision"
	BarColor     string `json:"bar_color"`      // e.g. "bg-emerald-500"
}

// StrategyDTO shows motivational stats for the mock exams page
type StrategyDTO struct {
	Message              string `json:"message"`                // "Mocks are the best way to identify weak points before the real deal."
	StudentsRegistered   int    `json:"students_registered"`    // e.g. 1240
	RegisteredLabel      string `json:"registered_label"`       // e.g. "+1,240 students registered this week"
	AvatarInitials       []string `json:"avatar_initials"`      // e.g. ["KP", "RN", "SS"]
}

