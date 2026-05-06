package teaching

import (
	"context"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
)

func TestSQLiteServiceSubmissionFlow(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:      "sqlite",
		SQLitePath:    t.TempDir() + "/teaching.db",
		MigrationsDir: "../../migrations",
		ReadyTimeout:  0,
	}
	pool, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()

	runner := migrate.NewRunner(pool, cfg.MigrationsDir)
	if _, err := runner.Up(context.Background()); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}

	repo := NewSQLiteRepository(pool)
	service := NewService(repo)

	bootstrapActor, err := NewActor("bootstrap-admin", []Role{RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := service.CreateUser(context.Background(), bootstrapActor, CreateUserInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
		Roles:       []string{"admin"},
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	adminActor, err := NewActor(admin.ID, []Role{RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	teacher, err := service.CreateUser(context.Background(), adminActor, CreateUserInput{
		Username:    "teacher1",
		DisplayName: "Teacher One",
		EmployeeNo:  "T001",
		Roles:       []string{"teacher"},
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create teacher: %v", err)
	}
	student, err := service.CreateUser(context.Background(), adminActor, CreateUserInput{
		Username:    "student1",
		DisplayName: "Student One",
		StudentNo:   "S001",
		Roles:       []string{"student"},
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create student: %v", err)
	}

	class, err := service.CreateClass(context.Background(), adminActor, CreateClassInput{
		Code:      "CLS001",
		Name:      "Class 1",
		GradeYear: 2026,
		Major:     "SE",
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create class: %v", err)
	}
	course, err := service.CreateCourse(context.Background(), adminActor, CreateCourseInput{
		Code:   "CRS001",
		Name:   "Course 1",
		Term:   "2026-spring",
		Status: "active",
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := service.AddCourseClass(context.Background(), adminActor, course.ID, class.ID, AuditEntry{}); err != nil {
		t.Fatalf("add course class: %v", err)
	}
	if err := service.AssignTeacher(context.Background(), adminActor, course.ID, AssignTeacherInput{
		TeacherID:  teacher.ID,
		Permission: "owner",
	}, AuditEntry{}); err != nil {
		t.Fatalf("assign teacher: %v", err)
	}
	if err := service.EnrollStudent(context.Background(), adminActor, course.ID, EnrollStudentInput{
		ClassID:   class.ID,
		StudentID: student.ID,
		Status:    "active",
	}, AuditEntry{}); err != nil {
		t.Fatalf("enroll student: %v", err)
	}

	teacherActor, err := NewActor(teacher.ID, []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	template, err := service.CreateRubricTemplate(context.Background(), teacherActor, CreateRubricTemplateInput{
		Name:   "Rubric 1",
		Scope:  "private",
		Status: "draft",
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	version, err := service.CreateRubricVersion(context.Background(), teacherActor, template.ID, CreateRubricVersionInput{
		WeightMode: string(WeightModeStrict100),
		Metrics: []MetricInput{
			{Code: "quality", Name: "Quality", WeightBPS: 6000, MaxScore: 100, SortOrder: 1},
			{Code: "docs", Name: "Docs", WeightBPS: 4000, MaxScore: 100, SortOrder: 2},
		},
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if _, err := service.PublishRubricVersion(context.Background(), teacherActor, version.Version.ID, AuditEntry{}); err != nil {
		t.Fatalf("publish version: %v", err)
	}
	experiment, err := service.CreateExperiment(context.Background(), teacherActor, course.ID, CreateExperimentInput{
		Title:           "Experiment 1",
		RubricVersionID: version.Version.ID,
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}
	if _, err := service.PublishExperiment(context.Background(), teacherActor, experiment.ID, AuditEntry{}); err != nil {
		t.Fatalf("publish experiment: %v", err)
	}

	studentActor, err := NewActor(student.ID, []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	access, err := repo.ExperimentSubmissionAccess(context.Background(), experiment.ID, student.ID)
	if err != nil {
		t.Fatalf("submission access: %v", err)
	}
	if access.Status != "published" || !access.Enrolled || access.CourseID != course.ID {
		t.Fatalf("unexpected submission access: %+v", access)
	}
	submission, err := service.CreateSubmission(context.Background(), studentActor, experiment.ID, CreateSubmissionInput{}, AuditEntry{})
	if err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if submission.ExperimentID != experiment.ID || submission.StudentID != student.ID {
		t.Fatalf("unexpected submission: %+v", submission)
	}
}
