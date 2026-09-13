// FIFO eviction keeps derived rendering data bounded during long editing sessions.
export class BoundedCache<K, V> extends Map<K, V> {
  constructor(private readonly capacity = 128) {
    super();
  }
  override set(key: K, value: V): this {
    if (!this.has(key) && this.size >= this.capacity) {
      const oldest = this.keys().next();
      if (!oldest.done) this.delete(oldest.value);
    }
    return super.set(key, value);
  }
}
