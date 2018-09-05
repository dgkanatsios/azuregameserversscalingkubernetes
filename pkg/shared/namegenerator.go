package shared

// // GenerateRandomName generates a random name with a prefix
// var GenerateRandomName func(prefix string) string

// func generateRandomName(prefix string) string {
// 	return prefix + "-" + randString(5)
// }

type RandomNameGenerator interface {
	GenerateName(prefix string) string
}

type FakeRandomNameGenerator interface {
	RandomNameGenerator
	SetName(name string)
}

type realRandomNameGenerator struct{}

func (*realRandomNameGenerator) GenerateName(prefix string) string {
	return prefix + "-" + randString(5)
}

func NewRealRandomNameGenerator() RandomNameGenerator {
	return &realRandomNameGenerator{}
}

type fakeRandomNameGenerator struct {
	name string
}

func (f *fakeRandomNameGenerator) SetName(p string) {
	f.name = p
}

func (f *fakeRandomNameGenerator) GenerateName(prefix string) string {
	return f.name
}

func NewFakeRandomNameGenerator() FakeRandomNameGenerator {
	return &fakeRandomNameGenerator{}
}
