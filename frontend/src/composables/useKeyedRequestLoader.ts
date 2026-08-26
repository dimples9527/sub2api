export function createKeyedRequestLoader<Params extends unknown[], Result, Key>(
  request: (...params: Params) => Promise<Result>,
  getKey: (...params: Params) => Key,
): (...params: Params) => Promise<Result> {
  let active: { key: Key; promise: Promise<Result> } | null = null

  return (...params: Params) => {
    const key = getKey(...params)
    if (active && Object.is(active.key, key)) {
      return active.promise
    }

    const promise = request(...params)
    active = { key, promise }
    void promise.then(
      () => {
        if (active?.promise === promise) active = null
      },
      () => {
        if (active?.promise === promise) active = null
      },
    )
    return promise
  }
}
