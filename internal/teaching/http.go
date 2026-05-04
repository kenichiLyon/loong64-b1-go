package teaching

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/httpx"
)

type HTTPDependencies struct {
	Service       *Service
	AppEnv        string
	Logger        *slog.Logger
	DevAuthBypass bool
}

type HTTPHandler struct {
	service       *Service
	appEnv        string
	logger        *slog.Logger
	devAuthBypass bool
}

func RegisterRoutes(mux *http.ServeMux, deps HTTPDependencies) {
	h := &HTTPHandler{service: deps.Service, appEnv: deps.AppEnv, logger: deps.Logger, devAuthBypass: deps.DevAuthBypass}
	mux.HandleFunc("GET /api/v1/me", h.me)
	mux.HandleFunc("GET /api/v1/admin/users", h.listUsers)
	mux.HandleFunc("POST /api/v1/admin/users", h.createUser)
	mux.HandleFunc("PUT /api/v1/admin/users/{userID}/roles", h.setUserRoles)
	mux.HandleFunc("POST /api/v1/admin/classes", h.createClass)
	mux.HandleFunc("POST /api/v1/admin/courses", h.createCourse)
	mux.HandleFunc("PUT /api/v1/admin/courses/{courseID}/classes", h.addCourseClass)
	mux.HandleFunc("PUT /api/v1/admin/courses/{courseID}/teachers", h.assignTeacher)
	mux.HandleFunc("PUT /api/v1/admin/courses/{courseID}/enrollments", h.enrollStudent)
	mux.HandleFunc("POST /api/v1/teacher/rubric-templates", h.createRubricTemplate)
	mux.HandleFunc("POST /api/v1/teacher/rubric-templates/{templateID}/versions", h.createRubricVersion)
	mux.HandleFunc("POST /api/v1/teacher/rubric-template-versions/{versionID}/publish", h.publishRubricVersion)
	mux.HandleFunc("POST /api/v1/teacher/courses/{courseID}/experiments", h.createExperiment)
	mux.HandleFunc("POST /api/v1/teacher/experiments/{experimentID}/publish", h.publishExperiment)
	mux.HandleFunc("POST /api/v1/student/experiments/{experimentID}/submissions", h.createSubmission)
	mux.HandleFunc("POST /api/v1/student/submissions/{submissionID}/artifacts", h.uploadArtifact)
	mux.HandleFunc("POST /api/v1/student/submissions/{submissionID}/artifact-links", h.createGitLinkArtifact)
	mux.HandleFunc("GET /api/v1/student/submissions/{submissionID}", h.getSubmissionDetail)
	mux.HandleFunc("GET /api/v1/teacher/experiments/{experimentID}/submissions", h.listExperimentSubmissions)
	mux.HandleFunc("GET /api/v1/teacher/submissions/{submissionID}", h.getSubmissionDetail)
	mux.HandleFunc("POST /api/v1/teacher/submissions/{submissionID}/evaluations/initial", h.createInitialEvaluation)
	mux.HandleFunc("GET /api/v1/teacher/submissions/{submissionID}/evaluations/latest", h.getLatestEvaluation)
	mux.HandleFunc("PUT /api/v1/teacher/submissions/{submissionID}/review", h.upsertTeacherReview)
	mux.HandleFunc("POST /api/v1/teacher/submissions/{submissionID}/review/publish", h.publishTeacherReview)
	mux.HandleFunc("GET /api/v1/teacher/submissions/{submissionID}/review", h.getTeacherReview)
	mux.HandleFunc("GET /api/v1/student/submissions/{submissionID}/review", h.getTeacherReview)
	mux.HandleFunc("GET /api/v1/teacher/submissions/{submissionID}/report", h.getSubmissionReport)
	mux.HandleFunc("GET /api/v1/student/submissions/{submissionID}/report", h.getSubmissionReport)
	mux.HandleFunc("POST /api/v1/teacher/submissions/{submissionID}/report-exports", h.createSubmissionReportExport)
	mux.HandleFunc("GET /api/v1/teacher/experiments/{experimentID}/reports/summary", h.getExperimentReportSummary)
	mux.HandleFunc("POST /api/v1/teacher/experiments/{experimentID}/report-exports", h.createExperimentSummaryExport)
	mux.HandleFunc("GET /api/v1/teacher/report-exports/{exportID}", h.getReportExport)
	mux.HandleFunc("GET /api/v1/teacher/report-exports/{exportID}/download", h.downloadReportExport)
}

func (h *HTTPHandler) me(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": actor.ID, "roles": actor.RoleValues()})
}

func (h *HTTPHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	users, err := h.service.ListUsers(r.Context(), actor, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (h *HTTPHandler) createUser(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateUserInput
	if !h.decode(w, r, &input) {
		return
	}
	user, err := h.service.CreateUser(r.Context(), actor, input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, user)
}

func (h *HTTPHandler) setUserRoles(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input struct {
		Roles []string `json:"roles"`
	}
	if !h.decode(w, r, &input) {
		return
	}
	if err := h.service.SetUserRoles(r.Context(), actor, r.PathValue("userID"), input.Roles, h.audit(r)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) createClass(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateClassInput
	if !h.decode(w, r, &input) {
		return
	}
	class, err := h.service.CreateClass(r.Context(), actor, input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, class)
}

func (h *HTTPHandler) createCourse(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateCourseInput
	if !h.decode(w, r, &input) {
		return
	}
	course, err := h.service.CreateCourse(r.Context(), actor, input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, course)
}

func (h *HTTPHandler) addCourseClass(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input struct {
		ClassID string `json:"class_id"`
	}
	if !h.decode(w, r, &input) {
		return
	}
	if err := h.service.AddCourseClass(r.Context(), actor, r.PathValue("courseID"), input.ClassID, h.audit(r)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) assignTeacher(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input AssignTeacherInput
	if !h.decode(w, r, &input) {
		return
	}
	if err := h.service.AssignTeacher(r.Context(), actor, r.PathValue("courseID"), input, h.audit(r)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) enrollStudent(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input EnrollStudentInput
	if !h.decode(w, r, &input) {
		return
	}
	if err := h.service.EnrollStudent(r.Context(), actor, r.PathValue("courseID"), input, h.audit(r)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) createRubricTemplate(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateRubricTemplateInput
	if !h.decode(w, r, &input) {
		return
	}
	template, err := h.service.CreateRubricTemplate(r.Context(), actor, input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, template)
}

func (h *HTTPHandler) createRubricVersion(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateRubricVersionInput
	if !h.decode(w, r, &input) {
		return
	}
	version, err := h.service.CreateRubricVersion(r.Context(), actor, r.PathValue("templateID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, version)
}

func (h *HTTPHandler) publishRubricVersion(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	version, err := h.service.PublishRubricVersion(r.Context(), actor, r.PathValue("versionID"), h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, version)
}

func (h *HTTPHandler) createExperiment(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateExperimentInput
	if !h.decode(w, r, &input) {
		return
	}
	experiment, err := h.service.CreateExperiment(r.Context(), actor, r.PathValue("courseID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, experiment)
}

func (h *HTTPHandler) publishExperiment(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	experiment, err := h.service.PublishExperiment(r.Context(), actor, r.PathValue("experimentID"), h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, experiment)
}

func (h *HTTPHandler) createSubmission(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateSubmissionInput
	if r.Body != nil && r.ContentLength != 0 {
		if !h.decode(w, r, &input) {
			return
		}
	}
	submission, err := h.service.CreateSubmission(r.Context(), actor, r.PathValue("experimentID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, submission)
}

func (h *HTTPHandler) uploadArtifact(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.service.maxUploadBytes+1024*1024)
	file, header, err := r.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeError(w, validationError("file exceeds max upload size"))
			return
		}
		h.writeError(w, validationError("multipart field file is required"))
		return
	}
	defer func() { _ = file.Close() }()
	artifact, err := h.service.UploadArtifact(r.Context(), actor, r.PathValue("submissionID"), ArtifactUploadInput{
		FileName:     header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		DeclaredKind: parseArtifactKindField(r.FormValue("artifact_kind")),
		Reader:       file,
	}, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, artifact)
}

func (h *HTTPHandler) createGitLinkArtifact(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateGitLinkInput
	if !h.decode(w, r, &input) {
		return
	}
	artifact, err := h.service.CreateGitLinkArtifact(r.Context(), actor, r.PathValue("submissionID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, artifact)
}

func (h *HTTPHandler) listExperimentSubmissions(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	submissions, err := h.service.ListSubmissionsForExperiment(r.Context(), actor, r.PathValue("experimentID"), limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": submissions})
}

func (h *HTTPHandler) getSubmissionDetail(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	detail, err := h.service.GetSubmissionDetail(r.Context(), actor, r.PathValue("submissionID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *HTTPHandler) createInitialEvaluation(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateInitialEvaluationInput
	if r.Body != nil && r.ContentLength != 0 {
		if !h.decode(w, r, &input) {
			return
		}
	}
	detail, err := h.service.CreateInitialEvaluation(r.Context(), actor, r.PathValue("submissionID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, detail)
}

func (h *HTTPHandler) getLatestEvaluation(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	detail, err := h.service.GetLatestEvaluation(r.Context(), actor, r.PathValue("submissionID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *HTTPHandler) upsertTeacherReview(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input UpsertTeacherReviewInput
	if !h.decode(w, r, &input) {
		return
	}
	detail, err := h.service.UpsertTeacherReview(r.Context(), actor, r.PathValue("submissionID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *HTTPHandler) publishTeacherReview(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input PublishTeacherReviewInput
	if !h.decode(w, r, &input) {
		return
	}
	detail, err := h.service.PublishTeacherReview(r.Context(), actor, r.PathValue("submissionID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *HTTPHandler) getTeacherReview(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	detail, err := h.service.GetTeacherReview(r.Context(), actor, r.PathValue("submissionID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *HTTPHandler) getSubmissionReport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	report, err := h.service.GetSubmissionReport(r.Context(), actor, r.PathValue("submissionID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, report)
}

func (h *HTTPHandler) getExperimentReportSummary(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	summary, err := h.service.GetExperimentReportSummary(r.Context(), actor, r.PathValue("experimentID"), limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (h *HTTPHandler) createSubmissionReportExport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateReportExportInput
	if r.Body != nil && r.ContentLength != 0 {
		if !h.decode(w, r, &input) {
			return
		}
	}
	export, err := h.service.CreateSubmissionReportExport(r.Context(), actor, r.PathValue("submissionID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, export)
}

func (h *HTTPHandler) createExperimentSummaryExport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var input CreateReportExportInput
	if r.Body != nil && r.ContentLength != 0 {
		if !h.decode(w, r, &input) {
			return
		}
	}
	export, err := h.service.CreateExperimentSummaryExport(r.Context(), actor, r.PathValue("experimentID"), input, h.audit(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, export)
}

func (h *HTTPHandler) getReportExport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	export, err := h.service.GetReportExport(r.Context(), actor, r.PathValue("exportID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, export)
}

func (h *HTTPHandler) downloadReportExport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.currentActor(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	file, err := h.service.OpenReportExport(r.Context(), actor, r.PathValue("exportID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	if file.Export.SHA256Hex != "" {
		w.Header().Set("X-Content-SHA256", file.Export.SHA256Hex)
	}
	http.ServeFile(w, r, file.Path)
}

func (h *HTTPHandler) currentActor(r *http.Request) (Actor, error) {
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	roleHeader := strings.TrimSpace(r.Header.Get("X-Actor-Roles"))
	if actorID == "" && roleHeader == "" && h.devAuthBypass && h.appEnv != "production" && isLocalRequest(r) {
		actorID = "dev-admin"
		roleHeader = "admin,teacher,student"
		if h.logger != nil {
			h.logger.Warn("using development auth bypass", "remote_addr", r.RemoteAddr)
		}
	}
	if actorID == "" {
		return Actor{}, unauthorizedError("X-Actor-ID is required")
	}
	parts := strings.FieldsFunc(roleHeader, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	roles, err := ParseRoleList(parts)
	if err != nil {
		return Actor{}, err
	}
	return NewActor(actorID, roles)
}

func (h *HTTPHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if h.logger != nil {
			h.logger.Warn("failed to decode JSON request body", "error", err)
		}
		if errors.Is(err, io.EOF) {
			h.writeError(w, validationError("request body is required"))
			return false
		}
		h.writeError(w, validationError("invalid JSON request body"))
		return false
	}
	return true
}

func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (h *HTTPHandler) audit(r *http.Request) AuditEntry {
	return AuditEntry{RequestID: r.Header.Get("X-Request-ID")}
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch ErrorKindOf(err) {
	case KindValidation:
		status = http.StatusBadRequest
	case KindUnauthorized:
		status = http.StatusUnauthorized
	case KindForbidden:
		status = http.StatusForbidden
	case KindNotFound:
		status = http.StatusNotFound
	case KindConflict:
		status = http.StatusConflict
	case KindUnavailable:
		status = http.StatusServiceUnavailable
	}
	message := err.Error()
	var appErr *Error
	if errors.As(err, &appErr) {
		message = appErr.Message
	}
	if h.logger != nil && status >= http.StatusInternalServerError {
		h.logger.Error("teaching api failed", "error", err)
	}
	httpx.WriteError(w, status, ErrorCodeOf(err), message)
}
