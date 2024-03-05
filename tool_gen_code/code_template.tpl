package {{.PackageName}}

type {{.StructName}} struct {
{{range $v := .KV}}{{print "\t"}}{{$v.Name}} {{$v.VType}}{{print "\t"}}`json:"{{$v.JsonName}}"`{{if $v.Comment}}{{print "\t"}}// {{$v.Comment}}{{end}}
{{end}}}

func (s *{{.StructName}}) GetStructName() string {
    return "{{.StructName}}"
}

{{range $v := .KV}}
func (s *{{$.StructName}}) Set{{$v.Name}}(setVal {{$v.VType}}) {
    s.{{$v.Name}} = setVal
}

func (s *{{$.StructName}}) Get{{$v.Name}}() {{$v.VType}} {
    return s.{{$v.Name}}
}
{{end}}