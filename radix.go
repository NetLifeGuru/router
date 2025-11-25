package router

type RadixNode struct {
	prefix   string
	children []*RadixNode
	isLeaf   bool
	entries  []RouteEntry
}

func (r *Router) insertNode(key string, entry RouteEntry) {
	r.insert(r.radixRoot, key, entry)
}

func longestCommonPrefixStr(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func matchPrefixWithStarStr(prefix, key string) (consumed int, ok bool) {
	i, j := 0, 0
	for i < len(prefix) {
		if prefix[i] == '*' {
			for j < len(key) && key[j] != '/' {
				j++
			}
			i++
			continue
		}
		if j >= len(key) || prefix[i] != key[j] {
			return 0, false
		}
		i++
		j++
	}
	return j, true
}

func (r *Router) insert(node *RadixNode, key string, entry RouteEntry) {
	for _, child := range node.children {
		if len(child.prefix) == 0 || key[0] != child.prefix[0] {
			continue
		}
		lcp := longestCommonPrefixStr(child.prefix, key)
		if lcp == 0 {
			continue
		}
		if lcp == len(child.prefix) && lcp == len(key) {
			child.isLeaf = true
			child.entries = append(child.entries, entry)
			return
		}
		if lcp < len(child.prefix) {
			newChild := &RadixNode{
				prefix:   child.prefix[lcp:],
				children: child.children,
				isLeaf:   child.isLeaf,
				entries:  child.entries,
			}
			child.prefix = child.prefix[:lcp]
			child.children = []*RadixNode{newChild}
			child.isLeaf = false
			child.entries = nil
		}
		if lcp < len(key) {
			r.insert(child, key[lcp:], entry)
		} else {
			child.isLeaf = true
			child.entries = append(child.entries, entry)
		}
		return
	}

	node.children = append(node.children, &RadixNode{prefix: key, isLeaf: true, entries: []RouteEntry{entry}})
}

func (r *Router) searchAll(key string, ctx *Context) bool {
	var dfs func(n *RadixNode, k string) bool
	dfs = func(n *RadixNode, k string) bool {
		if len(k) == 0 {
			if n.isLeaf {
				for _, e := range n.entries {
					ctx.Entries = append(ctx.Entries, e)
				}
				return true
			}
			return false
		}
		found := false

		for _, ch := range n.children {
			if len(ch.prefix) == 0 || ch.prefix[0] == '*' {
				continue
			}
			if k[0] != ch.prefix[0] {
				continue
			}
			if cons, ok := matchPrefixWithStarStr(ch.prefix, k); ok {
				if dfs(ch, k[cons:]) {
					found = true
				}
			}
		}

		for _, ch := range n.children {
			if len(ch.prefix) == 0 || ch.prefix[0] != '*' {
				continue
			}
			if cons, ok := matchPrefixWithStarStr(ch.prefix, k); ok {
				if dfs(ch, k[cons:]) {
					found = true
				}
			}
		}
		return found
	}
	return dfs(r.radixRoot, key)
}
