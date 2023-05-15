package hd

func (s *APISpec) Path(routePath, method string) *Path {
	r, ok := s.Paths[routePath]
	if !ok {
		r = map[string]*Path{}
	}

	if method == "" && len(r) > 0 {
		for k := range r {
			method = k
			break
		}
	}

	if method == "" {
		method = "post"
	}

	p, ok := r[method]
	if !ok {
		p = &Path{
			Responses: map[string]*Response{
				"200": {
					Description: "Success",
				},
			},
		}
	}

	r[method] = p
	s.Paths[routePath] = r

	return p
}

func (s *APISpec) SetPath(routePath, method string, apiPath *Path) {
	if method == "" {
		method = "post"
	}
	r := map[string]*Path{}
	r[method] = apiPath
	s.Paths[routePath] = r
}

func (p *Path) Parameter(name string) *Parameter {
	for _, param := range p.Parameters {
		if param.Name == name {
			return param
		}
	}

	param := &Parameter{
		Name:        name,
		Description: name,
	}
	p.Parameters = append(p.Parameters, param)
	return param
}

func (p *Path) Response(name string) *Response {
	for k, resp := range p.Responses {
		if k == name {
			return resp
		}
	}

	resp := &Response{
		Description: name,
	}
	p.Responses[name] = resp
	return resp
}
