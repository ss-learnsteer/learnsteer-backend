package lesson

import (
	"time"
)

// Subject represents an A/L academic subject (e.g. Biology, Physics, Chemistry)
type Subject struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `gorm:"not null" json:"name"`
	Code       string    `gorm:"not null;index" json:"code"`
	Stream     string    `gorm:"not null;index" json:"stream"`
	Medium     string    `gorm:"not null;default:'Sinhala';index" json:"medium"`
	Icon       string    `json:"icon"`
	Color      string    `json:"color"`
	OrderIndex int       `gorm:"default:0" json:"order_index"`

	Units []Unit `gorm:"foreignKey:SubjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"units,omitempty"`
}

// Unit represents a syllabus unit within a subject (e.g. Unit 01: Cell Biology)
type Unit struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	SubjectID   uint      `gorm:"index;not null" json:"subject_id"`
	UnitNumber  int       `gorm:"not null" json:"unit_number"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Color       string    `gorm:"default:'#059669'" json:"color"`
	OrderIndex  int       `gorm:"default:0" json:"order_index"`

	Lessons []Lesson `gorm:"foreignKey:UnitID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"lessons,omitempty"`
}

// Lesson represents an individual video lesson or learning module
type Lesson struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UnitID       uint      `gorm:"index;not null" json:"unit_id"`
	LessonNumber int       `gorm:"not null" json:"lesson_number"`
	Title        string    `gorm:"not null" json:"title"`
	Instructor   string    `json:"instructor"`
	VideoURL     string    `gorm:"not null" json:"video_url"`
	DurationMin  int       `gorm:"default:0" json:"duration_min"`
	LessonType   string    `gorm:"default:'Video'" json:"lesson_type"`
	IsVisible    bool      `gorm:"default:true" json:"is_visible"`
	IsLocked     bool      `gorm:"default:false" json:"is_locked"`
	OrderIndex   int       `gorm:"default:0" json:"order_index"`

	Notes     []LessonNote     `gorm:"foreignKey:LessonID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"notes,omitempty"`
	Resources []LessonResource `gorm:"foreignKey:LessonID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"resources,omitempty"`
}

// LessonNote represents a rich text / LaTeX structured notes section under a lesson
type LessonNote struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	LessonID        uint   `gorm:"index;not null" json:"lesson_id"`
	SectionType     string `gorm:"default:'text'" json:"section_type"`
	Title           string `json:"title"`
	ContentMarkdown string `gorm:"type:text;not null" json:"content_markdown"`
	ImageURL        string `json:"image_url,omitempty"`
	OrderIndex      int    `gorm:"default:0" json:"order_index"`
}

// LessonResource represents a downloadable supplementary file (PDF, DOCX, ZIP)
type LessonResource struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	LessonID uint   `gorm:"index;not null" json:"lesson_id"`
	Title    string `gorm:"not null" json:"title"`
	FileURL  string `gorm:"not null" json:"file_url"`
	FileType string `gorm:"default:'PDF'" json:"file_type"`
	FileSize string `json:"file_size"`
	Color    string `json:"color"`
}

// UserLessonProgress records a student's completion and watch progress for a lesson
type UserLessonProgress struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"index;not null" json:"user_id"`
	LessonID        uint       `gorm:"index;not null" json:"lesson_id"`
	IsCompleted     bool       `gorm:"default:false" json:"is_completed"`
	ProgressPercent int        `gorm:"default:0" json:"progress_percent"`
	CompletedAt     *time.Time `json:"completed_at"`
	LastWatchedAt   time.Time  `json:"last_watched_at"`
}

func (UserLessonProgress) TableName() string {
	return "user_lesson_progress"
}

// RevisionModule represents a unit revision package (short notes, essay guides)
type RevisionModule struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	SubjectID        uint      `gorm:"index;not null" json:"subject_id"`
	UnitID           *uint     `gorm:"index" json:"unit_id,omitempty"`
	Title            string    `gorm:"not null" json:"title"`
	TopicsCount      int       `gorm:"default:0" json:"topics_count"`
	ShortNotesPdfURL string    `json:"short_notes_pdf_url"`
	EssayGuidePdfURL string    `json:"essay_guide_pdf_url"`
	IsPopular        bool      `gorm:"default:false" json:"is_popular"`
}

func (RevisionModule) TableName() string {
	return "revision_modules"
}

// PastPaper represents an A/L past paper or provincial model paper
type PastPaper struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	SubjectID        uint      `gorm:"index;not null" json:"subject_id"`
	Year             int       `gorm:"not null;index" json:"year"`
	Badge            string    `gorm:"default:'A/L PAST PAPER'" json:"badge"`
	IsModel          bool      `gorm:"default:false" json:"is_model"`
	Title            string    `gorm:"not null" json:"title"`
	DurationMin      int       `gorm:"default:180" json:"duration_min"`
	Structure        string    `gorm:"default:'MCQ & Essay'" json:"structure"`
	PaperPdfURL      string    `json:"paper_pdf_url"`
	MarkingSchemeURL string    `json:"marking_scheme_url"`
}

func (PastPaper) TableName() string {
	return "past_papers"
}

// DTOs for API Responses

// SubjectSummaryDTO shapes the subject cards on /subjectlist
type SubjectSummaryDTO struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	Stream        string `json:"stream"`
	Medium        string `json:"medium"`
	Icon          string `json:"icon"`
	Color         string `json:"color"`
	TotalLessons  int    `json:"total_lessons"`
	TotalPapers   int    `json:"total_papers"`
	CompletionPct int    `json:"completion_pct"`
}

// UnitSummaryDTO shapes unit cards on /subjecthub and /lessonlist
type UnitSummaryDTO struct {
	ID               uint               `json:"id"`
	UnitNumber       int                `json:"unit_number"`
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Color            string             `json:"color"`
	TotalLessons     int                `json:"total_lessons"`
	CompletedLessons int                `json:"completed_lessons"`
	ProgressRatio    string             `json:"progress_ratio"` // e.g. "3/5"
	Lessons          []LessonSummaryDTO `json:"lessons,omitempty"`
}

// LessonSummaryDTO shapes lesson items inside units
type LessonSummaryDTO struct {
	ID           uint   `json:"id"`
	LessonNumber int    `json:"lesson_number"`
	Title        string `json:"title"`
	DurationMin  int    `json:"duration_min"`
	LessonType   string `json:"lesson_type"`
	IsCompleted  bool   `json:"is_completed"`
	ProgressPct  int    `json:"progress_pct"`
	StatusText   string `json:"status_text"` // e.g. "Review", "Watch Again", "65% Done", "Start"
	IsLocked     bool   `json:"is_locked"`
}

// LessonTabDTO represents a tab header on /lessonview
type LessonTabDTO struct {
	ID    string `json:"id"`              // "notes", "annotated", "qa", "resources"
	Label string `json:"label"`           // "Lesson Notes", "Annotated Program", "Q&A", "Resources"
	Count int    `json:"count,omitempty"` // e.g. 24 for Q&A, 3 for Resources
}

// LessonSubPointDTO represents sub-bullet points inside a phase section
type LessonSubPointDTO struct {
	Label string `json:"label"` // e.g. "G1 Phase:"
	Desc  string `json:"desc"`  // e.g. "Period before the synthesis of DNA."
}

// LessonNoteDetailDTO represents a structured note section
type LessonNoteDetailDTO struct {
	ID              uint                `json:"id"`
	Title           string              `json:"title"`
	ContentMarkdown string              `json:"content_markdown"`
	SectionType     string              `json:"section_type"` // "text", "phase", "tip"
	SubPoints       []LessonSubPointDTO `json:"sub_points,omitempty"`
	Icon            string              `json:"icon,omitempty"` // e.g. "🔬"
}

// LessonQADTO represents a Q&A thread for the lesson
type LessonQADTO struct {
	ID       uint   `json:"id"`
	User     string `json:"user"`     // e.g. "Amal P."
	Avatar   string `json:"avatar"`   // e.g. "AP"
	Question string `json:"question"` // e.g. "What is the difference between mitosis and meiosis?"
	Answer   string `json:"answer"`   // e.g. "Mitosis produces two identical diploid cells..."
	Time     string `json:"time"`     // e.g. "2 days ago"
	Likes    int    `json:"likes"`    // e.g. 12
}

// LessonResourceDetailDTO represents a downloadable attachment
type LessonResourceDetailDTO struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"` // e.g. "Mitosis Diagram Pack"
	FileSize string `json:"file_size"`
	FileType string `json:"file_type"` // "PDF", "DOCX"
	Color    string `json:"color"`     // "from-red-500 to-rose-600"
	FileURL  string `json:"file_url"`
}

// NextUpWidgetDTO represents the Next Up lesson widget in the sidebar
type NextUpWidgetDTO struct {
	LessonID       uint   `json:"lesson_id"`
	Unit           string `json:"unit"`            // e.g. "LESSON 4"
	Title          string `json:"title"`           // e.g. "Cytoplasmic Organelles & Functions"
	Duration       string `json:"duration"`        // e.g. "45 min"
	ResourcesCount int    `json:"resources_count"` // e.g. 8
	Icon           string `json:"icon"`            // e.g. "🧬"
}

// LessonUnitProgressDTO represents the unit progress box in the sidebar
type LessonUnitProgressDTO struct {
	Percentage       int    `json:"percentage"`        // e.g. 45
	CompletedLessons int    `json:"completed_lessons"` // e.g. 3
	TotalLessons     int    `json:"total_lessons"`     // e.g. 7
	Tip              string `json:"tip"`               // e.g. "Keep going! Focus on this unit to complete the Cell Biology unit by Sunday."
}

// LessonDetailDTO shapes the full /lessonview player page
type LessonDetailDTO struct {
	ID           uint                      `json:"id"`
	LessonNumber int                       `json:"lesson_number"`
	Title        string                    `json:"title"`
	UnitName     string                    `json:"unit_name"`
	SubjectName  string                    `json:"subject_name"`
	Instructor   string                    `json:"instructor"`
	VideoURL     string                    `json:"video_url"`
	DurationMin  int                       `json:"duration_min"`
	LessonType   string                    `json:"lesson_type"`
	IsCompleted  bool                      `json:"is_completed"`
	ProgressPct  int                       `json:"progress_pct"`
	Tabs         []LessonTabDTO            `json:"tabs"`
	Notes        []LessonNoteDetailDTO     `json:"notes"`
	QA           []LessonQADTO             `json:"qa"`
	Resources    []LessonResourceDetailDTO `json:"resources"`
	NextUp       NextUpWidgetDTO           `json:"next_up"`
	UnitProgress LessonUnitProgressDTO     `json:"unit_progress"`
}

// LessonListPageDTO represents the full payload for /lessonlist
type LessonListPageDTO struct {
	SubjectTitle    string                  `json:"subject_title"`
	SubjectDetails  string                  `json:"subject_details"`
	Units           []UnitSummaryDTO        `json:"units"`
	SubjectProgress LessonListProgressDTO   `json:"subject_progress"`
	IslandRankGoal  IslandRankGoalDTO       `json:"island_rank_goal"`
	UpcomingMock    UpcomingMockWidgetDTO   `json:"upcoming_mock"`
}

// LessonListProgressDTO represents the progress widget on /lessonlist
type LessonListProgressDTO struct {
	Percentage       int    `json:"percentage"`
	TotalLessons     int    `json:"total_lessons"`
	CompletedLessons int    `json:"completed_lessons"`
	StudyTime        string `json:"study_time"` // e.g. "148 hrs"
}

// IslandRankGoalDTO represents the rank goal widget
type IslandRankGoalDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpcomingMockWidgetDTO represents the upcoming mock widget on /lessonlist
type UpcomingMockWidgetDTO struct {
	Name    string `json:"name"`    // e.g. "Bio-Unit 01 Assesment"
	Details string `json:"details"` // e.g. "Starts in 2 days * 15.00PM"
	Extra   string `json:"extra"`   // e.g. "Master the expaned Unit 01 Lessons to unlock the practice simulation made early"
}

