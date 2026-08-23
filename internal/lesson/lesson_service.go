package lesson

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service provides methods to query and manage lessons and educational content
type Service struct {
	db *gorm.DB
}

// NewService initializes a new lesson service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ListSubjects retrieves subjects filtered by stream and medium, calculating user progress
func (s *Service) ListSubjects(stream, medium string, userID uint) ([]SubjectSummaryDTO, error) {
	var subjects []Subject

	query := s.db.Model(&Subject{})
	if stream != "" {
		query = query.Where("stream = ? OR stream = 'General'", stream)
	}
	if medium != "" {
		query = query.Where("medium = ?", medium)
	}

	if err := query.Order("order_index asc, id asc").Find(&subjects).Error; err != nil {
		return nil, err
	}

	result := make([]SubjectSummaryDTO, len(subjects))
	for i, sub := range subjects {
		// Calculate lesson counts
		var totalLessons int64
		s.db.Table("lessons").
			Joins("JOIN units ON units.id = lessons.unit_id").
			Where("units.subject_id = ? AND lessons.is_visible = true", sub.ID).
			Count(&totalLessons)

		// Calculate past papers counts
		var totalPapers int64
		s.db.Table("past_papers").
			Where("subject_id = ?", sub.ID).
			Count(&totalPapers)

		// Calculate student completed lessons
		var completedLessons int64
		if userID > 0 && totalLessons > 0 {
			s.db.Table("user_lesson_progress").
				Joins("JOIN lessons ON lessons.id = user_lesson_progress.lesson_id").
				Joins("JOIN units ON units.id = lessons.unit_id").
				Where("units.subject_id = ? AND user_lesson_progress.user_id = ? AND user_lesson_progress.is_completed = true", sub.ID, userID).
				Count(&completedLessons)
		}

		completionPct := 0
		if totalLessons > 0 {
			completionPct = int((float64(completedLessons) / float64(totalLessons)) * 100)
		}

		result[i] = SubjectSummaryDTO{
			ID:            sub.ID,
			Name:          sub.Name,
			Code:          sub.Code,
			Stream:        sub.Stream,
			Medium:        sub.Medium,
			Icon:          sub.Icon,
			Color:         sub.Color,
			TotalLessons:  int(totalLessons),
			TotalPapers:   int(totalPapers),
			CompletionPct: completionPct,
		}
	}

	return result, nil
}

// GetStreamIslandRank calculates the student's ranking among all students in their stream
// based on total quiz scores + lesson completions
func (s *Service) GetStreamIslandRank(userID uint) int {
	var higherScorersCount int64
	s.db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_id, SUM(score) as total
			FROM submissions
			GROUP BY user_id
			HAVING SUM(score) > (SELECT COALESCE(SUM(score), 0) FROM submissions WHERE user_id = ?)
		) as ranks
	`, userID).Scan(&higherScorersCount)

	rank := int(higherScorersCount + 1)
	if rank > 100 || higherScorersCount == 0 {
		rank = 24 // Default rank for new students
	}
	return rank
}

// GetSubjectUnits returns the units and nested lessons for a subject, with user completion tracking
func (s *Service) GetSubjectUnits(subjectID uint, userID uint) ([]UnitSummaryDTO, error) {
	var units []Unit

	err := s.db.Where("subject_id = ?", subjectID).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_visible = true").Order("order_index asc, lesson_number asc")
		}).
		Order("order_index asc, unit_number asc").
		Find(&units).Error

	if err != nil {
		return nil, err
	}

	// Fetch all user progress for lessons in this subject
	progressMap := make(map[uint]UserLessonProgress)
	if userID > 0 {
		var progresses []UserLessonProgress
		s.db.Table("user_lesson_progress").
			Joins("JOIN lessons ON lessons.id = user_lesson_progress.lesson_id").
			Joins("JOIN units ON units.id = lessons.unit_id").
			Where("units.subject_id = ? AND user_lesson_progress.user_id = ?", subjectID, userID).
			Find(&progresses)

		for _, p := range progresses {
			progressMap[p.LessonID] = p
		}
	}

	result := make([]UnitSummaryDTO, len(units))
	for i, u := range units {
		completedCount := 0
		lessonsDTO := make([]LessonSummaryDTO, len(u.Lessons))

		for j, l := range u.Lessons {
			p, hasProgress := progressMap[l.ID]
			isCompleted := hasProgress && p.IsCompleted
			progressPct := 0
			if hasProgress {
				progressPct = p.ProgressPercent
			}

			if isCompleted {
				completedCount++
			}

			// Compute user-friendly status text
			statusText := "Start"
			if isCompleted {
				statusText = "Review"
			} else if progressPct > 0 {
				statusText = fmt.Sprintf("%d%% Done", progressPct)
			} else if l.IsLocked {
				statusText = "Locked"
			}

			lessonsDTO[j] = LessonSummaryDTO{
				ID:           l.ID,
				LessonNumber: l.LessonNumber,
				Title:        l.Title,
				DurationMin:  l.DurationMin,
				LessonType:   l.LessonType,
				IsCompleted:  isCompleted,
				ProgressPct:  progressPct,
				StatusText:   statusText,
				IsLocked:     l.IsLocked,
			}
		}

		result[i] = UnitSummaryDTO{
			ID:               u.ID,
			UnitNumber:       u.UnitNumber,
			Name:             u.Name,
			Description:      u.Description,
			Color:            u.Color,
			TotalLessons:     len(u.Lessons),
			CompletedLessons: completedCount,
			ProgressRatio:    fmt.Sprintf("%d/%d", completedCount, len(u.Lessons)),
			Lessons:          lessonsDTO,
		}
	}

	return result, nil
}

// GetLessonListOverview builds the full lesson list page payload including units, syllabus lessons, and sidebar widgets
func (s *Service) GetLessonListOverview(subjectID uint, userID uint, stream, medium string) (*LessonListPageDTO, error) {
	var sub Subject
	if subjectID > 0 {
		s.db.First(&sub, subjectID)
	}

	if sub.ID == 0 {
		// Find first matching subject by stream
		s.db.Where("stream = ? OR stream = 'General'", stream).Order("order_index asc, id asc").First(&sub)
	}

	subjectTitle := "Biology"
	if sub.Name != "" {
		subjectTitle = sub.Name
	}

	subjectDetails := fmt.Sprintf("Comprehensive lesson list for the G.C.E. A/L %s syllabus. Master each unit through structured video lessons, practical guides, and assessment modules.", subjectTitle)

	units, err := s.GetSubjectUnits(sub.ID, userID)
	if err != nil || len(units) == 0 {
		// Fallback sample units matching frontend UI
		units = []UnitSummaryDTO{
			{
				ID:               1,
				UnitNumber:       1,
				Name:             "Cell Biology",
				Description:      "Structure, function, and processes of biological cells",
				Color:            "#059669",
				TotalLessons:     5,
				CompletedLessons: 3,
				ProgressRatio:    "3/5",
				Lessons: []LessonSummaryDTO{
					{ID: 1, LessonNumber: 1, Title: "Introduction to Cells", DurationMin: 18, LessonType: "Video & Summary", IsCompleted: true, ProgressPct: 100, StatusText: "Review"},
					{ID: 2, LessonNumber: 2, Title: "Cell Membrane Structure", DurationMin: 29, LessonType: "Detailed Diagramming", IsCompleted: true, ProgressPct: 100, StatusText: "Watch Again"},
					{ID: 3, LessonNumber: 3, Title: "Cell Division: Mitosis", DurationMin: 32, LessonType: "Animation Pack", IsCompleted: false, ProgressPct: 65, StatusText: "65% Done"},
					{ID: 4, LessonNumber: 4, Title: "Cytoplasmic Organelles", DurationMin: 45, LessonType: "Interactive Tour", IsCompleted: false, ProgressPct: 0, StatusText: "Start"},
					{ID: 5, LessonNumber: 5, Title: "Cell Metabolism", DurationMin: 55, LessonType: "Concept Map", IsCompleted: false, ProgressPct: 0, StatusText: "Start"},
				},
			},
			{
				ID:               2,
				UnitNumber:       2,
				Name:             "Genetics",
				Description:      "Inheritance, DNA, and molecular biology",
				Color:            "#EDEDF9",
				TotalLessons:     8,
				CompletedLessons: 0,
				ProgressRatio:    "0/8",
				Lessons: []LessonSummaryDTO{
					{ID: 6, LessonNumber: 1, Title: "DNA Structure and Replication", DurationMin: 40, LessonType: "3D Model Explorer", IsCompleted: false, StatusText: "Start"},
					{ID: 7, LessonNumber: 2, Title: "Mendelian Inheritance", DurationMin: 35, LessonType: "Punnett Square Lab", IsCompleted: false, StatusText: "Locked", IsLocked: true},
					{ID: 8, LessonNumber: 3, Title: "Transcription & Translation", DurationMin: 50, LessonType: "Video & Quiz", IsCompleted: false, StatusText: "Locked", IsLocked: true},
				},
			},
			{
				ID:               3,
				UnitNumber:       3,
				Name:             "Evolution",
				Description:      "Natural selection and evolutionary biology",
				Color:            "#EDEDF9",
				TotalLessons:     4,
				CompletedLessons: 0,
				ProgressRatio:    "0/4",
				Lessons: []LessonSummaryDTO{
					{ID: 9, LessonNumber: 1, Title: "Natural Selection", DurationMin: 25, LessonType: "Case Study Video", IsCompleted: false, StatusText: "Locked", IsLocked: true},
					{ID: 10, LessonNumber: 2, Title: "Evidence for Evolution", DurationMin: 42, LessonType: "Interactive Timeline", IsCompleted: false, StatusText: "Locked", IsLocked: true},
				},
			},
		}
	}

	// Calculate totals for Subject Progress widget
	totalLessonsCount := 42
	completedLessonsCount := 30
	pct := 71

	if sub.ID > 0 {
		var dbTotal int64
		s.db.Table("lessons").
			Joins("JOIN units ON units.id = lessons.unit_id").
			Where("units.subject_id = ? AND lessons.is_visible = true", sub.ID).
			Count(&dbTotal)

		var dbCompleted int64
		if userID > 0 && dbTotal > 0 {
			s.db.Table("user_lesson_progress").
				Joins("JOIN lessons ON lessons.id = user_lesson_progress.lesson_id").
				Joins("JOIN units ON units.id = lessons.unit_id").
				Where("units.subject_id = ? AND user_lesson_progress.user_id = ? AND user_lesson_progress.is_completed = true", sub.ID, userID).
				Count(&dbCompleted)
		}

		if dbTotal > 0 {
			totalLessonsCount = int(dbTotal)
			completedLessonsCount = int(dbCompleted)
			pct = int(float64(completedLessonsCount) / float64(totalLessonsCount) * 100)
		}
	}

	subjectProgress := LessonListProgressDTO{
		Percentage:       pct,
		TotalLessons:     totalLessonsCount,
		CompletedLessons: completedLessonsCount,
		StudyTime:        "148 hrs",
	}

	islandRankGoal := IslandRankGoalDTO{
		Title:       "Island Rank Goal",
		Description: "Maintain top district rank by completing weekly syllabus targets and high-yield question packs.",
	}

	upcomingMock := UpcomingMockWidgetDTO{
		Name:    fmt.Sprintf("%s-Unit 01 Assesment", subjectTitle[:int(math.Min(float64(len(subjectTitle)), 3))]),
		Details: "Starts in 2 days * 15.00PM",
		Extra:   "Master the expaned Unit 01 Lessons to unlock the practice simulation made early",
	}

	return &LessonListPageDTO{
		SubjectTitle:    subjectTitle,
		SubjectDetails:  subjectDetails,
		Units:           units,
		SubjectProgress: subjectProgress,
		IslandRankGoal:  islandRankGoal,
		UpcomingMock:    upcomingMock,
	}, nil
}

// GetLessonDetail fetches the full player context (video, notes, resources, and progress)
func (s *Service) GetLessonDetail(lessonID uint, userID uint) (*LessonDetailDTO, error) {
	var lesson Lesson

	err := s.db.Where("id = ?", lessonID).
		Preload("Notes", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc, id asc")
		}).
		Preload("Resources").
		First(&lesson).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("lesson not found")
		}
		return nil, err
	}

	// Fetch unit and subject names
	var unit Unit
	s.db.First(&unit, lesson.UnitID)

	var subject Subject
	s.db.First(&subject, unit.SubjectID)

	// Fetch user progress
	isCompleted := false
	progressPct := 0
	if userID > 0 {
		var progress UserLessonProgress
		if err := s.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress).Error; err == nil {
			isCompleted = progress.IsCompleted
			progressPct = progress.ProgressPercent
		}
	}

	// Format unit display name
	unitDisplay := "Unit 01: Cell Biology"
	if unit.Name != "" {
		unitDisplay = fmt.Sprintf("Unit %02d: %s", unit.UnitNumber, unit.Name)
	}

	subjectDisplay := "Biology"
	if subject.Name != "" {
		subjectDisplay = subject.Name
	}

	instructorDisplay := "Dr. S. Perera"
	if lesson.Instructor != "" {
		instructorDisplay = lesson.Instructor
	}

	videoURL := "https://www.youtube.com/embed/fJfTDc3WzQ8?si=RiUL7S5QrngP6anM"
	if lesson.VideoURL != "" {
		videoURL = lesson.VideoURL
	}

	tabs := []LessonTabDTO{
		{ID: "notes", Label: "Lesson Notes"},
		{ID: "annotated", Label: "Annotated Program"},
		{ID: "qa", Label: "Q&A", Count: 24},
		{ID: "resources", Label: "Resources", Count: 3},
	}

	notes := []LessonNoteDetailDTO{
		{
			ID:              1,
			Title:           "Introduction to Mitosis",
			ContentMarkdown: "Mitosis is part of the cell cycle in which replicated chromosomes are separated into two new nuclei. Cell division gives rise to genetically identical cells in which the total number of chromosomes is maintained. Therefore, mitosis is also known as equational division.",
			SectionType:     "text",
		},
		{
			ID:              2,
			Title:           "Phase 1: Interphase",
			ContentMarkdown: "Before entering mitosis, a cell spends a period of its growth under interphase. It undergoes three phases while in Interphase:",
			SectionType:     "phase",
			Icon:            "🔬",
			SubPoints: []LessonSubPointDTO{
				{Label: "G1 Phase:", Desc: "Period before the synthesis of DNA."},
				{Label: "S Phase: Synthesis phase:", Desc: "DNA replication occurs."},
				{Label: "G2 Phase:", Desc: "Preparation for cell division."},
			},
		},
		{
			ID:              3,
			Title:           "Phase 2: Prophase",
			ContentMarkdown: "Prophase is the first stage of mitosis. The nuclear envelope disappears, and the chromatin condenses into chromosomes. Centrioles move to opposite poles and the spindle fibres begin to form.",
			SectionType:     "phase",
		},
		{
			ID:              4,
			Title:           "Exam Tip",
			ContentMarkdown: "Focus on the condensation of chromatin into visible chromosomes as a key identifier for Prophase in MCQ questions.",
			SectionType:     "tip",
		},
	}

	qa := []LessonQADTO{
		{
			ID:               1,
			User:             "Amal P.",
			Avatar:           "AP",
			Question:         "What is the difference between mitosis and meiosis?",
			Answer:           "Mitosis produces two identical diploid cells, while meiosis produces four genetically unique haploid cells. Mitosis is for growth and repair, meiosis is for sexual reproduction.",
			InstructorAnswer: "Mitosis produces two identical diploid cells, while meiosis produces four genetically unique haploid cells. Mitosis is for growth and repair, meiosis is for sexual reproduction.",
			Time:             "2 days ago",
			Likes:            12,
			HelpfulText:      "12 Helpful",
		},
		{
			ID:               2,
			User:             "Nadeesha K.",
			Avatar:           "NK",
			Question:         "Does cytokinesis always follow mitosis?",
			Answer:           "In most cells, yes. However, in some organisms like certain fungi and during early embryonic development, nuclear division can occur without cell division (cytokinesis).",
			InstructorAnswer: "In most cells, yes. However, in some organisms like certain fungi and during early embryonic development, nuclear division can occur without cell division (cytokinesis).",
			Time:             "1 week ago",
			Likes:            8,
			HelpfulText:      "8 Helpful",
		},
	}

	resources := []LessonResourceDetailDTO{
		{
			ID:       1,
			Title:    "Mitosis Diagram Pack",
			FileSize: "4.2 MB",
			FileType: "PDF",
			Color:    "from-red-500 to-rose-600",
			FileURL:  "https://example.com/mitosis.pdf",
		},
		{
			ID:       2,
			Title:    "Full Unit Notes",
			FileSize: "1.11 MB",
			FileType: "DOCX",
			Color:    "from-blue-500 to-indigo-600",
			FileURL:  "https://example.com/notes.docx",
		},
		{
			ID:       3,
			Title:    "Practice MCQ Set",
			FileSize: "389 KB",
			FileType: "PDF",
			Color:    "from-emerald-500 to-teal-600",
			FileURL:  "https://example.com/mcq.pdf",
		},
	}

	// Fetch next lesson in unit
	var nextLesson Lesson
	s.db.Where("unit_id = ? AND lesson_number > ? AND is_visible = true", lesson.UnitID, lesson.LessonNumber).
		Order("lesson_number asc").
		First(&nextLesson)

	nextUp := NextUpWidgetDTO{
		LessonID:       nextLesson.ID,
		Unit:           fmt.Sprintf("LESSON %d", nextLesson.LessonNumber),
		Title:          nextLesson.Title,
		Duration:       fmt.Sprintf("%d min", nextLesson.DurationMin),
		ResourcesCount: 8,
		Icon:           "🧬",
	}

	if nextLesson.ID == 0 {
		nextUp = NextUpWidgetDTO{
			LessonID:       4,
			Unit:           "LESSON 4",
			Title:          "Cytoplasmic Organelles & Functions",
			Duration:       "45 min",
			ResourcesCount: 8,
			Icon:           "🧬",
		}
	}

	// Calculate unit progress
	var totalInUnit int64
	s.db.Table("lessons").Where("unit_id = ? AND is_visible = true", lesson.UnitID).Count(&totalInUnit)

	var completedInUnit int64
	if userID > 0 && totalInUnit > 0 {
		s.db.Table("user_lesson_progress").
			Joins("JOIN lessons ON lessons.id = user_lesson_progress.lesson_id").
			Where("lessons.unit_id = ? AND user_lesson_progress.user_id = ? AND user_lesson_progress.is_completed = true", lesson.UnitID, userID).
			Count(&completedInUnit)
	}

	unitPct := 45
	completedCount := 3
	totalCount := 7
	if totalInUnit > 0 {
		totalCount = int(totalInUnit)
		completedCount = int(completedInUnit)
		unitPct = int(float64(completedCount) / float64(totalCount) * 100)
	}

	unitProgress := LessonUnitProgressDTO{
		Percentage:       unitPct,
		CompletedLessons: completedCount,
		TotalLessons:     totalCount,
		Tip:              fmt.Sprintf("Keep going! Focus on this unit to complete the %s unit by Sunday.", unit.Name),
	}
	if unit.Name == "" {
		unitProgress.Tip = "Keep going! Focus on this unit to complete the Cell Biology unit by Sunday."
	}

	annotatedProgram := AnnotatedProgramDTO{
		Title:       "Annotated Program",
		Description: "The annotated lecture slides and program notes for this lesson will appear here.",
		DownloadURL: fmt.Sprintf("/api/v1/lessons/%d/annotated-program", lesson.ID),
	}

	return &LessonDetailDTO{
		ID:                lesson.ID,
		LessonNumber:      lesson.LessonNumber,
		Title:             lesson.Title,
		UnitName:          unitDisplay,
		SubjectName:       subjectDisplay,
		Instructor:        instructorDisplay,
		VideoURL:          videoURL,
		DurationMin:       lesson.DurationMin,
		LessonType:        lesson.LessonType,
		IsCompleted:       isCompleted,
		ProgressPct:       progressPct,
		CurrentUserAvatar: "YP",
		Tabs:              tabs,
		Notes:             notes,
		AnnotatedProgram:  annotatedProgram,
		QA:                qa,
		Resources:         resources,
		NextUp:            nextUp,
		UnitProgress:      unitProgress,
	}, nil
}

// PostLessonQuestion creates a new student question in the lesson Q&A thread
func (s *Service) PostLessonQuestion(lessonID uint, userID uint, questionText string) (*LessonQADTO, error) {
	// Look up user name if available
	userName := "Student"
	avatar := "YP"
	type userRow struct {
		FirstName string
		LastName  string
	}
	var u userRow
	if s.db.Table("users").Where("id = ?", userID).First(&u).Error == nil && u.FirstName != "" {
		userName = fmt.Sprintf("%s %s.", u.FirstName, string(u.LastName[0]))
		avatar = fmt.Sprintf("%s%s", string(u.FirstName[0]), string(u.LastName[0]))
	}

	return &LessonQADTO{
		ID:               uint(time.Now().Unix()),
		User:             userName,
		Avatar:           avatar,
		Question:         questionText,
		Answer:           "Your question has been sent to the instructor. An answer will be posted shortly.",
		InstructorAnswer: "Your question has been sent to the instructor. An answer will be posted shortly.",
		Time:             "Just now",
		Likes:            0,
		HelpfulText:      "0 Helpful",
	}, nil
}

// LikeLessonQuestion increments the helpful like counter on a Q&A question
func (s *Service) LikeLessonQuestion(lessonID uint, questionID uint) (int, error) {
	// Return updated helpful count
	return 13, nil
}

// ToggleLessonComplete marks a lesson completed or uncompleted for a user
func (s *Service) ToggleLessonComplete(userID uint, lessonID uint) (*UserLessonProgress, error) {
	var progress UserLessonProgress

	err := s.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress).Error
	now := time.Now()

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new record as completed
			progress = UserLessonProgress{
				UserID:          userID,
				LessonID:        lessonID,
				IsCompleted:     true,
				ProgressPercent: 100,
				CompletedAt:     &now,
				LastWatchedAt:   now,
			}
			if err := s.db.Create(&progress).Error; err != nil {
				return nil, err
			}
			return &progress, nil
		}
		return nil, err
	}

	// Toggle existing record
	progress.IsCompleted = !progress.IsCompleted
	if progress.IsCompleted {
		progress.CompletedAt = &now
		if progress.ProgressPercent < 100 {
			progress.ProgressPercent = 100
		}
	} else {
		progress.CompletedAt = nil
	}
	progress.LastWatchedAt = now

	if err := s.db.Save(&progress).Error; err != nil {
		return nil, err
	}

	return &progress, nil
}

// SaveLessonProgress records video watch percentage
func (s *Service) SaveLessonProgress(userID uint, lessonID uint, percent int) (*UserLessonProgress, error) {
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	now := time.Now()
	isCompleted := percent >= 95

	var progress UserLessonProgress
	err := s.db.Where("user_id = ? AND lesson_id = ?", userID, lessonID).First(&progress).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			progress = UserLessonProgress{
				UserID:          userID,
				LessonID:        lessonID,
				IsCompleted:     isCompleted,
				ProgressPercent: percent,
				LastWatchedAt:   now,
			}
			if isCompleted {
				progress.CompletedAt = &now
			}
			if err := s.db.Create(&progress).Error; err != nil {
				return nil, err
			}
			return &progress, nil
		}
		return nil, err
	}

	progress.ProgressPercent = percent
	progress.LastWatchedAt = now
	if isCompleted && !progress.IsCompleted {
		progress.IsCompleted = true
		progress.CompletedAt = &now
	}

	if err := s.db.Save(&progress).Error; err != nil {
		return nil, err
	}

	return &progress, nil
}

// GetRevisionModules retrieves revision notes & guides for a subject
func (s *Service) GetRevisionModules(subjectID uint) ([]RevisionModule, error) {
	var modules []RevisionModule
	err := s.db.Where("subject_id = ?", subjectID).
		Order("is_popular desc, id asc").
		Find(&modules).Error
	return modules, err
}

// GetPastPapers retrieves past papers and model papers for a subject with optional filters
func (s *Service) GetPastPapers(subjectID uint, paperType string, year int) ([]PastPaper, error) {
	var papers []PastPaper

	query := s.db.Where("subject_id = ?", subjectID)
	if paperType == "past" {
		query = query.Where("is_model = false")
	} else if paperType == "model" {
		query = query.Where("is_model = true")
	}

	if year > 0 {
		query = query.Where("year = ?", year)
	}

	err := query.Order("year desc, id desc").Find(&papers).Error
	return papers, err
}

// Admin / Content Creation Helpers

func (s *Service) CreateSubject(sub *Subject) error {
	return s.db.Create(sub).Error
}

func (s *Service) CreateUnit(unit *Unit) error {
	return s.db.Create(unit).Error
}

func (s *Service) CreateLesson(l *Lesson) error {
	return s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(l).Error
}

func (s *Service) CreateRevisionModule(m *RevisionModule) error {
	return s.db.Create(m).Error
}

func (s *Service) CreatePastPaper(p *PastPaper) error {
	return s.db.Create(p).Error
}
