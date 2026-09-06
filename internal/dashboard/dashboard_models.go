package dashboard

import "time"

// DashboardUserDTO contains the student's personal details and gamification metrics
type DashboardUserDTO struct {
	ID         uint   `json:"id"`
	StudentID  string `json:"student_id"`
	Nickname   string `json:"nickname"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	NIC        string `json:"nic"`
	Role       string `json:"role"`
	Stream     string `json:"stream"`
	Medium     string `json:"medium"`
	Batch      string `json:"batch"`
	DayStreak  int    `json:"day_streak"`
	IslandRank int    `json:"island_rank"`
	XP         int    `json:"xp"`
}

// SyllabusCoverageDTO represents the student's overall curriculum completion
type SyllabusCoverageDTO struct {
	Percentage       int    `json:"percentage"`
	CompletedLessons int    `json:"completed_lessons"`
	TotalLessons     int    `json:"total_lessons"`
	Status           string `json:"status"` // e.g. "Completed", "In Progress"
}

// NextMockExamDTO represents the upcoming recommended or eligible mock exam
type NextMockExamDTO struct {
	ID            uint   `json:"id"`
	ExamNumber    string `json:"exam_number"`    // e.g. "Mock Exam #04"
	Title         string `json:"title"`          // e.g. "Next: Organic Chemistry"
	Priority      string `json:"priority"`       // e.g. "High Priority"
	DurationMin   int    `json:"duration_min"`
	UnitsProgress string `json:"units_progress"` // e.g. "3/4 Units"
	ProgressPct   int    `json:"progress_pct"`   // e.g. 75
}

// SubjectProgressDTO represents progress for a single subject (Biology, Chemistry, etc.)
type SubjectProgressDTO struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Code             string `json:"code"`
	Percentage       int    `json:"percentage"`
	BarColor         string `json:"bar_color"` // e.g. "bg-emerald-500", "bg-violet-500", "bg-orange-400"
	IconBg           string `json:"icon_bg"`   // e.g. "bg-emerald-50", "bg-violet-50"
	Icon             string `json:"icon"`      // e.g. "🔬", "⚗️", "⚡", "📐"
	TotalLessons     int    `json:"total_lessons"`
	CompletedLessons int    `json:"completed_lessons"`
}

// SmartRevisionDTO provides targeted AI/recommendation alert for the student's weakest area
type SmartRevisionDTO struct {
	Topic       string `json:"topic"`        // e.g. "Cell Biology"
	SubTopic    string `json:"sub_topic"`    // e.g. "Mitochondria"
	AccuracyPct int    `json:"accuracy_pct"` // e.g. 42
	Message     string `json:"message"`      // e.g. "Focus on Cell Biology today. Your last test showed 42% accuracy in Mitochondria questions."
	ActionText  string `json:"action_text"`  // e.g. "Resume Lesson"
	LessonID    uint   `json:"lesson_id,omitempty"`
	UnitID      uint   `json:"unit_id,omitempty"`
}

// UpcomingDeadlineDTO represents an upcoming exam, mock paper, or assignment submission
type UpcomingDeadlineDTO struct {
	ID    uint      `json:"id"`
	Month string    `json:"month"` // e.g. "OCT"
	Day   int       `json:"day"`   // e.g. 18
	Title string    `json:"title"` // e.g. "Physics Unit 04 Submission"
	Time  string    `json:"time"`  // e.g. "8:00 PM"
	Place string    `json:"place"` // e.g. "Online", "Main Hall"
	Date  time.Time `json:"date"`
}

// DashboardResponseDTO represents the full dashboard response payload
type DashboardResponseDTO struct {
	User               DashboardUserDTO      `json:"user"`
	SyllabusCoverage   SyllabusCoverageDTO   `json:"syllabus_coverage"`
	NextMockExam       NextMockExamDTO       `json:"next_mock_exam"`
	SubjectProgress    []SubjectProgressDTO  `json:"subject_progress"`
	SmartRevision      SmartRevisionDTO      `json:"smart_revision"`
	UpcomingDeadlines  []UpcomingDeadlineDTO `json:"upcoming_deadlines"`
}
