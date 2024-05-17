package core

type Storage interface {
	Put(*Block) error
}

type Memorystore struct {
}

func NewMemorystore() *Memorystore {
	return nil
}

func (s *Memorystore) Put(b *Block) error {
	return nil
}
