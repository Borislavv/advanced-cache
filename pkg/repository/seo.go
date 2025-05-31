package repository

type Seo interface {
	PageData() ([]byte, error)
}

type SeoRepository struct{}

func NewSeo() *SeoRepository {
	return &SeoRepository{}
}

func (s *SeoRepository) PageData() ([]byte, error) {
	return []byte{}, nil
}
