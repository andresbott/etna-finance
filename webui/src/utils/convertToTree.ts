// buildTree takes a flat list of items and returns a tree object
// the flat list needs to be identified by id and parentId
export function buildTree(items: any[]) {
    if (!items || items.length === 0) return []

    const map = new Map<number, any>()
    const roots: any[] = []

    // create map
    items.forEach((item) => {
        map.set(item.id, { ...item, children: [] })
    })

    // build tree
    items.forEach((item) => {
        const node = map.get(item.id)
        if (item.parentId) {
            const parent = map.get(item.parentId)
            if (parent) {
                parent.children.push(node)
            }
        } else {
            roots.push(node)
        }
    })
    return roots
}

// buildTreeForTable takes a flat list of items and returns a tree compatible with PrimeVue TreeTable
// items must have 'id' and optional 'parentId'
export function buildTreeForTable(items: any[] | undefined) {
    if (!items || items.length === 0) return []

    const map = new Map<number, any>()
    const roots: any[] = []

    // create map with nodes structured for TreeTable
    items.forEach((item) => {
        map.set(item.id, {
            key: String(item.id), // unique key required by TreeTable
            data: { ...item }, // original item data
            children: undefined // initialize as undefined
        })
    })

    // build tree
    items.forEach((item) => {
        const node = map.get(item.id)
        if (item.parentId) {
            const parent = map.get(item.parentId)
            if (parent) {
                // initialize children if needed
                parent.children = parent.children || []
                parent.children.push(node)
            }
        } else {
            roots.push(node)
        }
    })

    return roots
}
