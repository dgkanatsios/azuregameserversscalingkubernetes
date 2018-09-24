package shared

// RandomNameGenerator is an interface for RandomNameGenerator structs
type RandomNameGenerator interface {
	GenerateName(prefix string) string
}

// FakeRandomNameGenerator is an interface for FakeRandomNameGenerator structs
type FakeRandomNameGenerator interface {
	RandomNameGenerator
	SetName(name string)
}

type realRandomNameGenerator struct{}

// GenerateName generates a random name that starts with the prefix and ends with 5 random characters
func (*realRandomNameGenerator) GenerateName(prefix string) string {
	return prefix + "-" + randString(RandStringSize)
}

// NewRealRandomNameGenerator returns a real (production) RandomNameGenerator
func NewRealRandomNameGenerator() RandomNameGenerator {
	return &realRandomNameGenerator{}
}

type fakeRandomNameGenerator struct {
	name string
}

// SetName sets the name that the fake random generator will produce
func (f *fakeRandomNameGenerator) SetName(p string) {
	f.name = p
}

// GenerateName returns a fake random name (previously defined by the SetName method)
func (f *fakeRandomNameGenerator) GenerateName(prefix string) string {
	return f.name
}

// NewFakeRandomNameGenerator returns a new FakeRandomNameGenerator
func NewFakeRandomNameGenerator() FakeRandomNameGenerator {
	return &fakeRandomNameGenerator{}
}
