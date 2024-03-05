package result

type Struct2 struct {
	Id   int    `json:"id"`   // id。
	Name string `json:"name"` // 名字。
}

func (s *Struct2) GetStructName() string {
	return "Struct2"
}

func (s *Struct2) SetId(setVal int) {
	s.Id = setVal
}

func (s *Struct2) GetId() int {
	return s.Id
}

func (s *Struct2) SetName(setVal string) {
	s.Name = setVal
}

func (s *Struct2) GetName() string {
	return s.Name
}
