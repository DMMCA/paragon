package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/handler"
	"github.com/kcarretto/paragon/ent"
	"github.com/kcarretto/paragon/graphql/generated"
	"github.com/kcarretto/paragon/graphql/resolve"
	"github.com/kcarretto/paragon/pkg/event"

	"go.uber.org/zap"
)

// Service provides HTTP handlers for the GraphQL schema.
type Service struct {
	Log    *zap.Logger
	Graph  *ent.Client
	Events event.Publisher
}

// HandleGraphQL initializes and returns a new GraphQL API handler.
func (svc *Service) HandleGraphQL() http.Handler {
	resolver := &resolve.Resolver{
		Log:    svc.Log.Named("resolver"),
		Graph:  svc.Graph,
		Events: svc.Events,
	}
	config := generated.Config{Resolvers: resolver}
	schema := generated.NewExecutableSchema(config)

	return handler.GraphQL(schema)
}

// HandlePlayground initializes and returns a new GraphQL Playground handler.
func (svc *Service) HandlePlayground() http.Handler {
	return handler.Playground("GraphQL", "/graphql")
}

// HTTP registers http handlers for a GraphQL API.
func (svc *Service) HTTP(router *http.ServeMux) {
	router.Handle("/graphql", svc.HandleGraphQL())
	router.Handle("/graphiql", svc.HandlePlayground())
}
