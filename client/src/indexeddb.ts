// GoSPA IndexedDB Persistence
// Provides persistent state storage using IndexedDB for large datasets
// Complements localStorage which is limited to ~5MB

/**
 * IndexedDB configuration
 */
export interface IndexedDBConfig {
  /** Database name */
  dbName?: string;
  /** Database version */
  version?: number;
  /** Store name for state */
  storeName?: string;
  /** Enable auto-cleanup of old entries */
  autoCleanup?: boolean;
  /** Maximum age for entries in milliseconds (default: 7 days) */
  maxAge?: number;
}

/**
 * Stored entry structure
 */
interface StoredEntry<T = unknown> {
  key: string;
  value: T;
  timestamp: number;
  expiresAt?: number;
}

/**
 * IndexedDB persistence manager
 */
export class IndexedDBPersistence {
  private db: IDBDatabase | null = null;
  private config: Required<IndexedDBConfig>;
  private initPromise: Promise<void> | null = null;

  constructor(config: IndexedDBConfig = {}) {
    this.config = {
      dbName: config.dbName ?? "gospa-state",
      version: config.version ?? 1,
      storeName: config.storeName ?? "state",
      autoCleanup: config.autoCleanup ?? true,
      maxAge: config.maxAge ?? 7 * 24 * 60 * 60 * 1000, // 7 days
    };
  }

  /**
   * Initialize the IndexedDB database
   */
  private init(): Promise<void> {
    if (this.initPromise) return this.initPromise;

    this.initPromise = new Promise((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new Error("IndexedDB not available"));
        return;
      }

      const request = indexedDB.open(this.config.dbName, this.config.version);

      request.onerror = () => {
        reject(
          new Error(`Failed to open IndexedDB: ${request.error?.message}`),
        );
      };

      request.onsuccess = () => {
        this.db = request.result;
        if (
          typeof process !== "undefined" &&
          process.env?.NODE_ENV !== "production"
        ) {
          console.log(
            `[GoSPA IndexedDB] Database opened: ${this.config.dbName}`,
          );
        }

        // Setup cleanup on success
        if (this.config.autoCleanup) {
          this.cleanup().catch(console.error);
        }

        resolve();
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;

        // Create object store if it doesn't exist
        if (!db.objectStoreNames.contains(this.config.storeName)) {
          const store = db.createObjectStore(this.config.storeName, {
            keyPath: "key",
          });
          store.createIndex("timestamp", "timestamp", { unique: false });
          store.createIndex("expiresAt", "expiresAt", { unique: false });
          if (
            typeof process !== "undefined" &&
            process.env?.NODE_ENV !== "production"
          ) {
            console.log(
              `[GoSPA IndexedDB] Created store: ${this.config.storeName}`,
            );
          }
        }
      };
    });

    return this.initPromise;
  }

  /**
   * Get a value from IndexedDB
   */
  async get<T>(key: string): Promise<T | null> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readonly",
      );
      const store = transaction.objectStore(this.config.storeName);
      const request = store.get(key);

      request.onerror = () => {
        reject(
          new Error(`Failed to get key ${key}: ${request.error?.message}`),
        );
      };

      request.onsuccess = () => {
        const entry = request.result as StoredEntry<T> | undefined;

        if (!entry) {
          resolve(null);
          return;
        }

        // Check if entry has expired
        if (entry.expiresAt && Date.now() > entry.expiresAt) {
          // Entry expired, delete it
          this.delete(key).catch(console.error);
          resolve(null);
          return;
        }

        resolve(entry.value);
      };
    });
  }

  /**
   * Set a value in IndexedDB
   */
  async set<T>(key: string, value: T, ttl?: number): Promise<void> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const entry: StoredEntry<T> = {
        key,
        value,
        timestamp: Date.now(),
        expiresAt: ttl ? Date.now() + ttl : undefined,
      };

      const transaction = this.db.transaction(
        this.config.storeName,
        "readwrite",
      );
      const store = transaction.objectStore(this.config.storeName);
      const request = store.put(entry);

      request.onerror = () => {
        reject(
          new Error(`Failed to set key ${key}: ${request.error?.message}`),
        );
      };

      request.onsuccess = () => {
        resolve();
      };
    });
  }

  /**
   * Delete a value from IndexedDB
   */
  async delete(key: string): Promise<void> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readwrite",
      );
      const store = transaction.objectStore(this.config.storeName);
      const request = store.delete(key);

      request.onerror = () => {
        reject(
          new Error(`Failed to delete key ${key}: ${request.error?.message}`),
        );
      };

      request.onsuccess = () => {
        resolve();
      };
    });
  }

  /**
   * Get all keys from IndexedDB
   */
  async keys(): Promise<string[]> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readonly",
      );
      const store = transaction.objectStore(this.config.storeName);
      const request = store.getAllKeys();

      request.onerror = () => {
        reject(new Error(`Failed to get keys: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        resolve(request.result as string[]);
      };
    });
  }

  /**
   * Clear all entries from IndexedDB
   */
  async clear(): Promise<void> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readwrite",
      );
      const store = transaction.objectStore(this.config.storeName);
      const request = store.clear();

      request.onerror = () => {
        reject(new Error(`Failed to clear store: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        if (
          typeof process !== "undefined" &&
          process.env?.NODE_ENV !== "production"
        ) {
          console.log(
            `[GoSPA IndexedDB] Cleared store: ${this.config.storeName}`,
          );
        }
        resolve();
      };
    });
  }

  /**
   * Clean up expired entries
   */
  async cleanup(): Promise<number> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readwrite",
      );
      const store = transaction.objectStore(this.config.storeName);
      const index = store.index("expiresAt");
      const now = Date.now();
      let deletedCount = 0;

      // Open cursor on expiresAt index
      const request = index.openCursor(IDBKeyRange.upperBound(now));

      request.onerror = () => {
        reject(new Error(`Failed to cleanup: ${request.error?.message}`));
      };

      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          cursor.delete();
          deletedCount++;
          cursor.continue();
        } else {
          if (
            deletedCount > 0 &&
            typeof process !== "undefined" &&
            process.env?.NODE_ENV !== "production"
          ) {
            console.log(
              `[GoSPA IndexedDB] Cleaned up ${deletedCount} expired entries`,
            );
          }
          resolve(deletedCount);
        }
      };
    });
  }

  /**
   * Get database size estimate
   */
  async getSize(): Promise<{ entries: number; bytes: number }> {
    await this.init();

    return new Promise((resolve, reject) => {
      if (!this.db) {
        reject(new Error("Database not initialized"));
        return;
      }

      const transaction = this.db.transaction(
        this.config.storeName,
        "readonly",
      );
      const store = transaction.objectStore(this.config.storeName);
      const countRequest = store.count();
      let entries = 0;

      countRequest.onerror = () => {
        reject(
          new Error(`Failed to count entries: ${countRequest.error?.message}`),
        );
      };

      countRequest.onsuccess = () => {
        entries = countRequest.result;

        // Estimate size (rough approximation)
        const getAllRequest = store.getAll();
        getAllRequest.onerror = () => {
          // If getAll fails, just return count
          resolve({ entries, bytes: 0 });
        };

        getAllRequest.onsuccess = () => {
          const data = getAllRequest.result;
          const bytes = new Blob([JSON.stringify(data)]).size;
          resolve({ entries, bytes });
        };
      };
    });
  }

  /**
   * Close the database connection
   */
  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
      this.initPromise = null;
      if (
        typeof process !== "undefined" &&
        process.env?.NODE_ENV !== "production"
      ) {
        console.log(`[GoSPA IndexedDB] Database closed: ${this.config.dbName}`);
      }
    }
  }

  /**
   * Delete the entire database
   */
  async deleteDatabase(): Promise<void> {
    this.close();

    return new Promise((resolve, reject) => {
      const request = indexedDB.deleteDatabase(this.config.dbName);

      request.onerror = () => {
        reject(
          new Error(`Failed to delete database: ${request.error?.message}`),
        );
      };

      request.onsuccess = () => {
        if (
          typeof process !== "undefined" &&
          process.env?.NODE_ENV !== "production"
        ) {
          console.log(
            `[GoSPA IndexedDB] Database deleted: ${this.config.dbName}`,
          );
        }
        resolve();
      };
    });
  }
}

/**
 * Create an IndexedDB persistence manager
 */
export function createIndexedDBPersistence(
  config?: IndexedDBConfig,
): IndexedDBPersistence {
  return new IndexedDBPersistence(config);
}

/**
 * Global IndexedDB persistence instance
 */
let globalPersistence: IndexedDBPersistence | null = null;

/**
 * Get or create the global IndexedDB persistence instance
 */
export function getIndexedDBPersistence(
  config?: IndexedDBConfig,
): IndexedDBPersistence {
  if (!globalPersistence) {
    globalPersistence = new IndexedDBPersistence(config);
  }
  return globalPersistence;
}

/**
 * Destroy the global IndexedDB persistence instance
 */
export function destroyIndexedDBPersistence(): void {
  if (globalPersistence) {
    globalPersistence.close();
    globalPersistence = null;
  }
}
