import { writable } from "svelte/store"

export type ToastType = "success" | "error" | "warning" | "info"

export interface Toast {
  id: number
  type: ToastType
  title: string
  message?: string
  duration?: number
}

let nextId = 1

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([])

  function add(type: ToastType, title: string, message?: string, duration = 4000) {
    const id = nextId++
    update((toasts) => {
      // Max 5 toasts at a time
      const next = [...toasts, { id, type, title, message, duration }]
      return next.length > 5 ? next.slice(next.length - 5) : next
    })

    if (duration > 0) {
      setTimeout(() => remove(id), duration)
    }

    return id
  }

  function remove(id: number) {
    update((toasts) => toasts.filter((t) => t.id !== id))
  }

  function success(title: string, message?: string, duration?: number) {
    return add("success", title, message, duration)
  }

  function error(title: string, message?: string, duration?: number) {
    return add("error", title, message, duration ?? 6000)
  }

  function warning(title: string, message?: string, duration?: number) {
    return add("warning", title, message, duration)
  }

  function info(title: string, message?: string, duration?: number) {
    return add("info", title, message, duration)
  }

  return { subscribe, add, remove, success, error, warning, info }
}

export const toast = createToastStore()
