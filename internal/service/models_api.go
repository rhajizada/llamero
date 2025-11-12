package service

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rhajizada/llamero/internal/models"
)

// ListModels returns LLM-compatible model metadata aggregated from backends.
func (s *Service) ListModels(ctx context.Context) (models.ModelList, error) {
	modelMap, err := s.collectModels(ctx)
	if err != nil {
		return models.ModelList{}, err
	}
	list := models.ModelList{Object: "list"}
	for _, model := range modelMap {
		list.Data = append(list.Data, model)
	}
	sort.Slice(list.Data, func(i, j int) bool {
		return list.Data[i].ID < list.Data[j].ID
	})
	return list, nil
}

// GetModel returns metadata for a single model.
func (s *Service) GetModel(ctx context.Context, id string) (models.Model, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return models.Model{}, &Error{Code: http.StatusBadRequest, Message: "model id is required"}
	}
	modelMap, err := s.collectModels(ctx)
	if err != nil {
		return models.Model{}, err
	}
	if model, ok := modelMap[id]; ok {
		return model, nil
	}
	return models.Model{}, &Error{Code: http.StatusNotFound, Message: "model not found"}
}

func (s *Service) collectModels(ctx context.Context) (map[string]models.Model, error) {
	statuses, err := s.store.ListBackends(ctx)
	if err != nil {
		return nil, &Error{Code: http.StatusInternalServerError, Message: "failed to load models", Err: err}
	}
	modelMap := make(map[string]models.Model)
	for _, status := range statuses {
		now := status.UpdatedAt.Unix()
		owned := status.ID
		if owned == "" {
			owned = defaultModelOwner
		}
		if len(status.ModelMeta) == 0 {
			for _, name := range status.Models {
				addModel(modelMap, name, now, owned)
			}
			continue
		}
		for _, meta := range status.ModelMeta {
			created := meta.CreatedAt.Unix()
			if created == 0 {
				created = now
			}
			owner := meta.OwnedBy
			if strings.TrimSpace(owner) == "" {
				owner = owned
			}
			addModel(modelMap, meta.Name, created, owner)
		}
	}
	return modelMap, nil
}

func addModel(dest map[string]models.Model, name string, created int64, ownedBy string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	existing, ok := dest[name]
	if ok {
		if created != 0 && (existing.Created == 0 || created < existing.Created) {
			existing.Created = created
		}
		if existing.OwnedBy == "" && ownedBy != "" {
			existing.OwnedBy = ownedBy
		}
		dest[name] = existing
		return
	}
	if created == 0 {
		created = timeNowUnix()
	}
	if strings.TrimSpace(ownedBy) == "" {
		ownedBy = defaultModelOwner
	}
	dest[name] = models.Model{
		ID:      name,
		Object:  "model",
		Created: created,
		OwnedBy: ownedBy,
	}
}

func timeNowUnix() int64 { return time.Now().Unix() }
