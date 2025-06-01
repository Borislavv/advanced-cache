package repository

type Seo interface {
	PageData() ([]byte, error)
}

type SeoRepository struct{}

func NewSeo() *SeoRepository {
	return &SeoRepository{}
}

func (s *SeoRepository) PageData() ([]byte, error) {
	return []byte("In Go, a pool of byte slices can be implemented using sync.Pool to reuse byte slices and reduce memory allocations, which can improve performance. Here's how it works:\n1. Creating a Pool\nA sync.Pool is created with a New function that defines how to create a new byte slice when the pool is empty.\nThe New function typically returns an empty slice with a specific capacity.In Go, a pool of byte slices can be implemented using sync.Pool to reuse byte slices and reduce memory allocations, which can improve performance. Here's how it works:\n1. Creating a Pool\nA sync.Pool is created with a New function that defines how to create a new byte slice when the pool is empty.\nThe New function typically returns an empty slice with a specific capacity.In Go, a pool of byte slices can be implemented using sync.Pool to reuse byte slices and reduce memory allocations, which can improve performance. Here's how it works:\n1. Creating a Pool\nA sync.Pool is created with a New function that defines how to create a new byte slice when the pool is empty.\nThe New function typically returns an empty slice with a specific capacity."), nil
}
