package handy

import (
	"errors"
	"strings"
)

var (
	ErrRouteNotFound      = errors.New("Router not found")
	ErrRouteAlreadyExists = errors.New("Route already exists")
	ErrCannotAppendRoute  = errors.New("Cannot append route")
	ErrOnlyOneWildcard    = errors.New("Only one wildcard is allowed in this level")
)

type node struct {
	name             string
	handler          Handler
	isWildcard       bool
	hasChildWildcard bool
	parent           *node
	children         map[string]*node
	wildcardName     string
}

type Router struct {
	root    *node
	current *node
}

func NewRouter() *Router {
	r := new(Router)
	root := new(node)
	root.children = make(map[string]*node)
	r.root = root
	r.current = r.root
	return r
}

func isWildcard(l string) bool {
	return l[0] == '{' && l[len(l)-1] == '}'
}

func cleanWildcard(l string) string {
	return l[1 : len(l)-1]
}

func (r *Router) nodeExists(n string) (*node, bool) {
	v, ok := r.current.children[n]
	if !ok && r.current.hasChildWildcard {
		// looking for wildcard
		v, ok = r.current.children[r.current.wildcardName]
	}

	return v, ok
}

func (r *Router) AppendRoute(uri string, h Handler) error {
	uri = strings.TrimSpace(uri)

	appended := false
	tokens := strings.Split(uri, "/")
	for k, v := range tokens {
		if v == "" {
			continue
		}

		if n, ok := r.nodeExists(v); ok {
			if len(tokens)-1 == k {
				return ErrRouteAlreadyExists
			}

			r.current = n
			appended = true
			continue
		}

		n := new(node)
		n.children = make(map[string]*node)

		// only one child wildcard per node
		if isWildcard(v) {
			if r.current.hasChildWildcard {
				return ErrOnlyOneWildcard
			}

			n.isWildcard = true
			r.current.wildcardName = v
			r.current.hasChildWildcard = true
		}

		n.name = v
		n.parent = r.current
		r.current.children[n.name] = n
		r.current = n
		appended = true
	}

	if r.current != r.root {
		r.current.handler = h
		r.current = r.root // reset
	}

	if appended == false {
		return ErrCannotAppendRoute
	}

	return nil

}

func (n *node) findChild(name string) *node {
	v, ok := n.children[name]
	if !ok && n.hasChildWildcard {
		// looking for wildcard
		v, ok = n.children[n.wildcardName]
	}

	return v
}

type RouteMatch struct {
	URIVars map[string]string
	Handler Handler
}

// This method rebuilds a route based on a given URI
func (r *Router) Match(uri string) (*RouteMatch, error) {
	rt := new(RouteMatch)
	rt.URIVars = make(map[string]string)

	current := r.current
	for _, v := range strings.Split(uri, "/") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		n := current.findChild(v)
		if n == nil {
			return rt, ErrRouteNotFound
		}

		if n.isWildcard {
			rt.URIVars[cleanWildcard(n.name)] = v
		}

		current = n
	}

	if current.handler == nil {
		return rt, ErrRouteNotFound
	}

	rt.Handler = current.handler
	return rt, nil
}