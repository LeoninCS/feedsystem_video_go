import { ref } from 'vue'
import { useToastStore } from '../stores/toast'

export function useVideoPlayer(scrollerRef: ReturnType<typeof ref<HTMLDivElement | null>>) {
  const toast = useToastStore()
  const muted = ref(true)
  const activeIndex = ref(0)
  const videoMap = new Map<number, HTMLVideoElement>()

  function getScrollerHeight() {
    return scrollerRef.value?.clientHeight ?? 0
  }

  function setVideoRef(id: number, el: HTMLVideoElement | null) {
    if (el) {
      el.muted = muted.value
      videoMap.set(id, el)
    } else {
      videoMap.delete(id)
    }
  }

  function scrollToIndex(idx: number, totalItems: number) {
    const el = scrollerRef.value
    if (!el) return
    const h = getScrollerHeight()
    if (!h) return
    const next = Math.max(0, Math.min(idx, Math.max(0, totalItems - 1)))
    el.scrollTo({ top: next * h, behavior: 'smooth' })
  }

  let scrollRaf = 0
  function onScroll() {
    if (!scrollerRef.value) return
    if (scrollRaf) return
    scrollRaf = window.requestAnimationFrame(() => {
      scrollRaf = 0
      const el = scrollerRef.value
      if (!el) return
      const h = el.clientHeight
      if (!h) return
      const idx = Math.round(el.scrollTop / h)
      if (idx !== activeIndex.value) activeIndex.value = idx
    })
  }

  async function playActive(activeItemId: number | undefined) {
    if (!activeItemId) return
    for (const [id, v] of videoMap.entries()) {
      if (id === activeItemId) continue
      v.pause()
    }
    const video = videoMap.get(activeItemId)
    if (!video) return
    video.muted = muted.value
    try {
      await video.play()
    } catch {
      /* ignore autoplay errors */
    }
  }

  function toggleMute() {
    muted.value = !muted.value
    for (const v of videoMap.values()) v.muted = muted.value
    toast.info(muted.value ? '已静音' : '已取消静音')
  }

  function togglePlayPause(activeItemId: number | undefined) {
    if (!activeItemId) return
    const video = videoMap.get(activeItemId)
    if (!video) return
    if (video.paused) void video.play()
    else video.pause()
  }

  return { muted, activeIndex, videoMap, setVideoRef, scrollToIndex, onScroll, playActive, toggleMute, togglePlayPause }
}
