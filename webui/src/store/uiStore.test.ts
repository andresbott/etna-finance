import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from './uiStore'

describe('uiStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('initial state', () => {
    it('has drawer hidden by default', () => {
      const store = useUiStore()
      expect(store.isDrawerVisible).toBe(false)
    })

    it('has secondary drawer hidden by default', () => {
      const store = useUiStore()
      expect(store.isSecondaryDrawerVisible).toBe(false)
    })
  })

  describe('main drawer', () => {
    it('openDrawer sets isDrawerVisible to true', () => {
      const store = useUiStore()
      store.openDrawer()
      expect(store.isDrawerVisible).toBe(true)
    })

    it('closeDrawer sets isDrawerVisible to false', () => {
      const store = useUiStore()
      store.openDrawer()
      store.closeDrawer()
      expect(store.isDrawerVisible).toBe(false)
    })

    it('closeDrawer when already closed stays false', () => {
      const store = useUiStore()
      store.closeDrawer()
      expect(store.isDrawerVisible).toBe(false)
    })

    it('toggleDrawer opens when closed', () => {
      const store = useUiStore()
      store.toggleDrawer()
      expect(store.isDrawerVisible).toBe(true)
    })

    it('toggleDrawer closes when open', () => {
      const store = useUiStore()
      store.openDrawer()
      store.toggleDrawer()
      expect(store.isDrawerVisible).toBe(false)
    })

    it('toggleDrawer twice returns to original state', () => {
      const store = useUiStore()
      store.toggleDrawer()
      store.toggleDrawer()
      expect(store.isDrawerVisible).toBe(false)
    })
  })

  describe('secondary drawer', () => {
    it('openSecondaryDrawer sets isSecondaryDrawerVisible to true', () => {
      const store = useUiStore()
      store.openSecondaryDrawer()
      expect(store.isSecondaryDrawerVisible).toBe(true)
    })

    it('closeSecondaryDrawer sets isSecondaryDrawerVisible to false', () => {
      const store = useUiStore()
      store.openSecondaryDrawer()
      store.closeSecondaryDrawer()
      expect(store.isSecondaryDrawerVisible).toBe(false)
    })

    it('closeSecondaryDrawer when already closed stays false', () => {
      const store = useUiStore()
      store.closeSecondaryDrawer()
      expect(store.isSecondaryDrawerVisible).toBe(false)
    })

    it('toggleSecondaryDrawer opens when closed', () => {
      const store = useUiStore()
      store.toggleSecondaryDrawer()
      expect(store.isSecondaryDrawerVisible).toBe(true)
    })

    it('toggleSecondaryDrawer closes when open', () => {
      const store = useUiStore()
      store.openSecondaryDrawer()
      store.toggleSecondaryDrawer()
      expect(store.isSecondaryDrawerVisible).toBe(false)
    })

    it('secondary drawer is independent of main drawer', () => {
      const store = useUiStore()
      store.openDrawer()
      store.openSecondaryDrawer()
      expect(store.isDrawerVisible).toBe(true)
      expect(store.isSecondaryDrawerVisible).toBe(true)

      store.closeDrawer()
      expect(store.isDrawerVisible).toBe(false)
      expect(store.isSecondaryDrawerVisible).toBe(true)
    })
  })

  describe('checkScreenWidth', () => {
    const originalInnerWidth = window.innerWidth

    afterEach(() => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: originalInnerWidth
      })
    })

    it('opens drawer when screen width is >= 1024', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1024
      })
      const store = useUiStore()
      store.initUi()
      expect(store.isDrawerVisible).toBe(true)
    })

    it('opens drawer when screen width is larger than 1024', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1920
      })
      const store = useUiStore()
      store.initUi()
      expect(store.isDrawerVisible).toBe(true)
    })

    it('closes drawer when screen width is < 1024', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1023
      })
      const store = useUiStore()
      store.openDrawer()
      store.initUi()
      expect(store.isDrawerVisible).toBe(false)
    })

    it('closes drawer on mobile width', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375
      })
      const store = useUiStore()
      store.initUi()
      expect(store.isDrawerVisible).toBe(false)
    })
  })

  describe('initUi and cleanupUi', () => {
    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('initUi calls checkScreenWidth and adds resize listener', () => {
      const addSpy = vi.spyOn(window, 'addEventListener')
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1024
      })

      const store = useUiStore()
      store.initUi()

      expect(store.isDrawerVisible).toBe(true)
      expect(addSpy).toHaveBeenCalledWith('resize', expect.any(Function))
    })

    it('cleanupUi removes resize listener', () => {
      const removeSpy = vi.spyOn(window, 'removeEventListener')

      const store = useUiStore()
      store.cleanupUi()

      expect(removeSpy).toHaveBeenCalledWith('resize', expect.any(Function))
    })

    it('initUi then cleanupUi properly manages the listener', () => {
      const addSpy = vi.spyOn(window, 'addEventListener')
      const removeSpy = vi.spyOn(window, 'removeEventListener')

      const store = useUiStore()
      store.initUi()
      store.cleanupUi()

      expect(addSpy).toHaveBeenCalledTimes(1)
      expect(removeSpy).toHaveBeenCalledTimes(1)
    })
  })
})
