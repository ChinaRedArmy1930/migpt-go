package vector_stores

type VectorStore interface {
	CollectionName() string
	Load() error
	SimilaritySearch(msg string)
}
