# /new-provider

Ajoute un nouveau provider LLM dans Veil.

## Usage
```
/new-provider <nom> <base_url>
```

## Exemple
```
/new-provider groq https://api.groq.com
```

## Ce que cette commande fait

1. Crée `internal/provider/<nom>.go` en implémentant l'interface Provider
2. Enregistre le provider dans `internal/provider/registry.go`
3. Ajoute les variables d'env dans `.env.example`
4. Crée le fichier de test `internal/provider/<nom>_test.go`

## Interface à implémenter

```go
type Provider interface {
    Complete(ctx context.Context, req *models.CompletionRequest) (*models.CompletionResponse, error)
    Stream(ctx context.Context, req *models.CompletionRequest) (<-chan models.StreamChunk, error)
    Health(ctx context.Context) error
    Name() string
    CostPer1kTokens() (input float64, output float64)
}
```

## Checklist après génération

- [ ] Vérifier le format de requête du provider (OpenAI-compatible ?)
- [ ] Vérifier le format de réponse (streaming SSE ?)
- [ ] Ajouter le coût réel dans CostPer1kTokens()
- [ ] Ajouter la clé API dans .env.example
- [ ] Tester avec `go test ./internal/provider/...`
- [ ] Documenter dans CLAUDE.md section "Marges par provider"