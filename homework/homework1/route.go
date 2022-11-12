package web

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
func (r *router) addRoute(method string, path string, handler HandleFunc) {
	root, ok := r.trees[method]
	log.Println("current path is ", path)
	if !ok {
		root = NewRootNode(nil)
		r.trees[method] = root
	}
	// 1. 检查 path 是否合法
	if err := checkPath(path, checkPathEmpty, startWithSlash, endWithSlash, noDuplicateSlash); err != nil {
		panic(err.Error())
	}
	// 2. 按照 / 分割 path
	paths := strings.Split(path, "/")
	// 3. 从 root 开始，逐级查找或者创建节点
	for _, p := range paths {
		if p == "" {
			continue
		}
		root = root.childOrCreate(p)
	}
	root.handler = handler

}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	path = strings.Trim(path, "/")

	if path == "" {
		return &matchInfo{
			root,
			nil,
		}, true
	}

	paths := strings.Split(path, "/")

	log.Printf("current path is %s\n", path)

	for _, p := range paths {
		child, ok := root.childOf(p)
		if !ok {
			return nil, false
		}
		root = child
	}

	return &matchInfo{
		root,
		nil}, true
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ nodeType

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

func NewRootNode(handle HandleFunc) *node {
	return &node{
		path:    "/",
		handler: handle,
	}
}

func NewStaticNode() *node {
	return &node{
		typ: nodeTypeStatic,
	}
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {
	if n.children == nil {
		return nil, false
	}
	child, ok := n.children[path]
	return child, ok
}

// childOrCreate 查找子节点，
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) childOrCreate(path string) *node {
	res, ok := n.childOf(path)
	if ok {
		return res
	}
	var strategy = buildStrategy(path)

	if strategy == nil {
		if n.children == nil {
			n.children = make(map[string]*node)
		}
		res = NewStaticNode()
		n.children[path] = res
		return res
	}

	return strategy(path, n)
}

type routerMatchStrategy func(path string, n *node) *node

func buildStrategy(path string) routerMatchStrategy {
	if path == "*" {
		return wildcardRouterStrategy
	}

	if path[0] == ':' {
		return paramRouterStrategy
	}

	if path[0] == '(' && path[len(path)-1] == ')' {
		return regRouterStrategy
	}

	return nil
}

func paramRouterStrategy(path string, n *node) *node {
	if n.paramChild == nil {
		n.paramChild = &node{
			typ:       nodeTypeParam,
			path:      path,
			paramName: path[1:],
		}
	}
	return n.paramChild
}

func regRouterStrategy(path string, n *node) *node {
	if n.regChild == nil {
		regExpr, err := regexp.Compile(path[1 : len(path)-1])
		if err != nil {
			panic(err)
		}
		n.regChild = &node{
			typ:       nodeTypeReg,
			path:      path,
			regExpr:   regExpr,
			paramName: path[1:],
		}
	}
	return n.regChild
}

func wildcardRouterStrategy(path string, n *node) *node {
	if n.starChild == nil {
		n.starChild = &node{
			typ:  nodeTypeAny,
			path: path,
		}
	}
	return n.starChild
}

func staticRouterStrategy(path string, n *node) *node {
	n.children = make(map[string]*node)
	n.children[path] = NewStaticNode()
	return n.children[path]
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func newMatchInfo(n *node, pathParams map[string]string) *matchInfo {
	return &matchInfo{
		n,
		pathParams,
	}
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}

// 检测path是否合法
func checkPath(path string, rules ...ruleFunc) error {
	for _, r := range rules {
		if err := r(path); err != nil {
			return err
		}
	}
	return nil
}

type ruleFunc func(path string) error

func checkPathEmpty(path string) error {
	if path == "" {
		return fmt.Errorf("web: 路由是空字符串")
	}
	return nil
}

func startWithSlash(path string) error {
	if path[0] != '/' {
		return fmt.Errorf("web: 路由必须以 / 开头")
	}
	return nil
}

func endWithSlash(path string) error {
	if path[len(path)-1] == '/' && len(path) > 1 {
		return fmt.Errorf("web: 路由不能以 / 结尾")
	}
	return nil
}

func noDuplicateSlash(path string) error {
	if strings.Contains(path, "//") {
		return fmt.Errorf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path)
	}
	return nil
}
