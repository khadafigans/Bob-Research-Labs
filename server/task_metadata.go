package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Autumn-27/artex/db"
)

const (
	maxTaskNameRunes           = 200
	maxTaskMetadataRequestSize = 16 << 10
)

// 작업 메타데이터 수정 API 가 사용자에게 돌려주는 오류 응답 문구. JSON 필드명
// (name·pinned)은 원문 그대로 두고, 사람이 읽는 메시지만 한국어로 둔다.
const (
	errTaskMetaRequestTooLarge = "요청 본문이 너무 큽니다"
	errTaskMetaNoFields        = "At least one of name or pinned must be provided"
	errTaskMetaNameEmpty       = "Task name cannot be empty"
	// errTaskMetaNameTooLongFmt 는 fmt.Sprintf 로 상한을 채우는 형식 문자열이다.
	errTaskMetaNameTooLongFmt = "Task name can be at most %d characters"
)

// updateTaskMetadata changes list-only task metadata. It intentionally does not
// touch lifecycle state, scheduling, or the task's immutable description/goal.
func (s *Server) updateTaskMetadata(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if _, ok := s.m.Task(taskID); !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTaskMetadataRequestSize)
	var request struct {
		Name   *string `json:"name"`
		Pinned *bool   `json:"pinned"`
	}
	if err := decode(r, &request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeErr(w, http.StatusRequestEntityTooLarge, errTaskMetaRequestTooLarge)
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	if request.Name == nil && request.Pinned == nil {
		writeErr(w, http.StatusBadRequest, errTaskMetaNoFields)
		return
	}
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			writeErr(w, http.StatusBadRequest, errTaskMetaNameEmpty)
			return
		}
		if utf8.RuneCountInString(name) > maxTaskNameRunes {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf(errTaskMetaNameTooLongFmt, maxTaskNameRunes))
			return
		}
		request.Name = &name
	}
	task, err := s.m.UpdateTaskMetadata(taskID, db.TaskPatch{Name: request.Name, Pinned: request.Pinned})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, taskDTO(task, s.resolvedTaskStatus(task)))
}
