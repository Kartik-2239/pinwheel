package proxy

import (
	"context"
	"fmt"

	"github.com/Kartik-2239/pinwheel/internal/db"
)

func Router(store *db.Store, context context.Context, model string, key string) (db.Model, error) {

	// modelname := ensureModelName(model)

	models, err := store.GetModelFromName(context, model, key)
	for _, model := range models {
		fmt.Println(model.Model, model.Provider.Name, model.Provider.BaseURL, model.Provider.EnvKey)
	}
	fmt.Println(len(models))
	if err != nil {
		return db.Model{}, err
	}
	return models[0], nil
}
