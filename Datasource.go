package main

type Datasource struct {
	Name string
	path string
}

func NewDatasource(name string) Datasource {
	return Datasource{name, ""}
}

func (self *Datasource) Init(name string) {
	self.Name = name
}
