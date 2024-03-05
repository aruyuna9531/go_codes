package result

type Struct1 struct {
	Id       int    `json:"id"`       // 它的id
	Id2      int    `json:"id2"`      // 它的第2个id
	Name     string `json:"name"`     // 它的名字
	IntArray []int  `json:"intArray"` // 它的数据组
}

func (s *Struct1) GetStructName() string {
	return "Struct1"
}

func (s *Struct1) SetId(setVal int) {
	s.Id = setVal
}

func (s *Struct1) GetId() int {
	return s.Id
}

func (s *Struct1) SetId2(setVal int) {
	s.Id2 = setVal
}

func (s *Struct1) GetId2() int {
	return s.Id2
}

func (s *Struct1) SetName(setVal string) {
	s.Name = setVal
}

func (s *Struct1) GetName() string {
	return s.Name
}

func (s *Struct1) SetIntArray(setVal []int) {
	s.IntArray = setVal
}

func (s *Struct1) GetIntArray() []int {
	return s.IntArray
}
