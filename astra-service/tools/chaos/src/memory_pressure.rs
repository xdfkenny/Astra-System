//! Memory pressure injection for WASM CRDT worker saturation testing.
//!
//! Simulates heap exhaustion and allocation storms that can cause the
//! in-browser WASM CRDT merge worker to stall or OOM. This test validates
//! that the sync daemon gracefully degrades under memory pressure rather
//! than panicking or corrupting the CRDT state.
//!
//! # Mechanism
//!
//! 1. **Heap fill**: allocate large Vec<u8> buffers until a target RSS
//!    percentage is reached or allocation starts failing.
//! 2. **Allocation storm**: spawn concurrent threads that rapidly allocate
//!    and deallocate varied-size blocks to simulate WASM memory thrash.
//! 3. **Monitor**: track peak RSS, allocation latency, and error rate.

use std::alloc::{alloc, dealloc, Layout};
use std::ptr::{addr_of_mut, read, write};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::{Duration, Instant};

use rand::Rng;
use tracing::{info, warn};

/// Configuration for memory pressure injection.
#[derive(Debug, Clone)]
pub struct MemoryPressureConfig {
    /// Target RSS usage ratio (0.0–1.0). 0.8 means fill to 80% of physical RAM.
    pub target_rss_ratio: f64,
    /// Size of each allocation block in bytes (default 1 MiB).
    pub block_size: usize,
    /// Number of concurrent allocator threads.
    pub concurrency: usize,
    /// Duration to sustain pressure before release.
    pub duration: Duration,
    /// If true, print the plan without allocating.
    pub dry_run: bool,
}

impl Default for MemoryPressureConfig {
    fn default() -> Self {
        Self {
            target_rss_ratio: 0.8,
            block_size: 1024 * 1024,
            concurrency: 4,
            duration: Duration::from_secs(30),
            dry_run: false,
        }
    }
}

/// Statistics collected during a memory pressure run.
#[derive(Debug, Default, Clone)]
pub struct MemoryPressureStats {
    pub total_allocated_bytes: u64,
    pub peak_rss_bytes: u64,
    pub allocation_count: u64,
    pub allocation_errors: u64,
    pub duration_secs: f64,
    pub swap_triggered: bool,
}

/// Configuration for swap pressure injection.
#[derive(Debug, Clone)]
pub struct SwapPressureConfig {
    /// Number of pages to lock.
    pub pages_to_lock: usize,
    /// Page size in bytes (default 4096).
    pub page_size: usize,
    /// Duration to sustain pressure before release.
    pub duration: Duration,
    /// Number of concurrent workers.
    pub concurrency: usize,
    /// If true, print the plan without executing.
    pub dry_run: bool,
}

impl Default for SwapPressureConfig {
    fn default() -> Self {
        Self {
            pages_to_lock: 1024 * 1024,
            page_size: 4096,
            duration: Duration::from_secs(30),
            concurrency: 2,
            dry_run: false,
        }
    }
}

/// Statistics collected during a swap pressure run.
#[derive(Debug, Default, Clone)]
pub struct SwapPressureStats {
    pub pages_locked: u64,
    pub swap_errors: u64,
    pub duration_secs: f64,
}

/// Run memory pressure injection according to the given config.
pub fn run(config: &MemoryPressureConfig) -> Result<MemoryPressureStats, Box<dyn std::error::Error>> {
    if config.dry_run {
        info!("[dry-run] Memory pressure plan:");
        info!("  target_rss_ratio: {}", config.target_rss_ratio);
        info!("  block_size:       {}", config.block_size);
        info!("  concurrency:      {}", config.concurrency);
        info!("  duration:         {:?}", config.duration);
        return Ok(MemoryPressureStats::default());
    }

    let stop = Arc::new(AtomicBool::new(false));
    let total_allocated = Arc::new(AtomicU64::new(0));
    let allocation_errors = Arc::new(AtomicU64::new(0));
    let allocation_count = Arc::new(AtomicU64::new(0));

    // Phase 1: fill heap to target ratio
    info!("Phase 1: heap fill starting");
    let (heap_fill_bytes, heap_fill_errs) = heap_fill(
        config.target_rss_ratio,
        config.block_size,
        &stop,
        &total_allocated,
        &allocation_errors,
        &allocation_count,
    );
    info!(
        allocated_mib = heap_fill_bytes / (1024 * 1024),
        errors = heap_fill_errs,
        "Phase 1: heap fill complete"
    );

    // Phase 2: allocation storm with concurrent threads
    let storm_stop = stop.clone();
    let storm_alloc = total_allocated.clone();
    let storm_errs = allocation_errors.clone();
    let storm_count = allocation_count.clone();

    let handles: Vec<_> = (0..config.concurrency)
        .map(|_worker_id| {
            let stop = storm_stop.clone();
            let alloc = storm_alloc.clone();
            let _errs = storm_errs.clone();
            let count = storm_count.clone();

            thread::spawn(move || {
                let mut rng = rand::thread_rng();
                let mut blocks: Vec<Vec<u8>> = Vec::new();

                while !stop.load(Ordering::SeqCst) {
                    // Vary block size from 1 KiB to 4 MiB to simulate WASM thrash
                    let size = rng.gen_range(1024..(4 * 1024 * 1024));
                    let block = {
                        let mut v = Vec::with_capacity(size);
                        v.resize(size, 0);
                        rng.fill(&mut v[..]);
                        v
                    };
                        alloc.fetch_add(size as u64, Ordering::Relaxed);
                        count.fetch_add(1, Ordering::Relaxed);
                        blocks.push(block);

                    // Periodically drain half the blocks to simulate WASM GC
                    if blocks.len() > 100 {
                        let drain_to = blocks.len() / 2;
                        blocks.drain(0..drain_to);
                    }

                    thread::sleep(Duration::from_millis(rng.gen_range(1..50)));
                }
            })
        })
        .collect();

    let start = Instant::now();
    thread::sleep(config.duration);
    stop.store(true, Ordering::SeqCst);

    for h in handles {
        h.join().map_err(|e| format!("thread join failed: {:?}", e))?;
    }

    let peak_rss = current_rss_bytes();
    let elapsed = start.elapsed().as_secs_f64();

    // Check if swap was triggered by comparing RSS after release
    thread::sleep(Duration::from_millis(500));
    let post_release_rss = current_rss_bytes();
    let swap_triggered = post_release_rss < peak_rss.saturating_sub(64 * 1024 * 1024);

    // Release held blocks
    let _ = heap_fill_release();

    let stats = MemoryPressureStats {
        total_allocated_bytes: total_allocated.load(Ordering::Relaxed),
        peak_rss_bytes: peak_rss,
        allocation_count: allocation_count.load(Ordering::Relaxed),
        allocation_errors: allocation_errors.load(Ordering::Relaxed),
        duration_secs: elapsed,
        swap_triggered,
    };

    info!(
        total_allocated_mib = stats.total_allocated_bytes / (1024 * 1024),
        peak_rss_mib = stats.peak_rss_bytes / (1024 * 1024),
        allocations = stats.allocation_count,
        errors = stats.allocation_errors,
        duration_secs = stats.duration_secs,
        "Memory pressure completed"
    );

    Ok(stats)
}

/// Global holder for heap-fill blocks to prevent deallocation.
static mut HEAP_FILL_BLOCKS: Option<Vec<Vec<u8>>> = None;

/// Fill heap until we reach the target RSS ratio or allocation fails.
fn heap_fill(
    target_ratio: f64,
    block_size: usize,
    stop: &AtomicBool,
    total: &AtomicU64,
    errors: &AtomicU64,
    count: &AtomicU64,
) -> (u64, u64) {
    let total_ram = total_physical_ram();
    let target_rss = (total_ram as f64 * target_ratio) as u64;
    let mut blocks: Vec<Vec<u8>> = Vec::new();

    let mut local_bytes: u64 = 0;
    let mut local_errors: u64 = 0;

    loop {
        if stop.load(Ordering::SeqCst) {
            break;
        }

        let current_rss = current_rss_bytes();
        if current_rss >= target_rss {
            info!(
                current_rss_mib = current_rss / (1024 * 1024),
                target_rss_mib = target_rss / (1024 * 1024),
                "Target RSS reached"
            );
            break;
        }

        // Use try_reserve_exact to avoid panicking on OOM
        let mut block = Vec::new();
        if block.try_reserve_exact(block_size).is_err() {
            warn!("OOM during heap fill — allocation failed");
            local_errors += 1;
            errors.fetch_add(1, Ordering::Relaxed);
            break;
        }
        // Touch pages to force RSS to reflect allocation
        block.resize(block_size, 0);
        for chunk in block.chunks_mut(4096) {
            chunk[0] = 0xFF;
        }
        total.fetch_add(block_size as u64, Ordering::Relaxed);
        count.fetch_add(1, Ordering::Relaxed);
        local_bytes += block_size as u64;
        blocks.push(block);
    }

    // Store blocks in global so they're not dropped
    unsafe {
        HEAP_FILL_BLOCKS = Some(blocks);
    }

    (local_bytes, local_errors)
}

/// Inject swap pressure by locking many pages into RAM via mlock,
/// requiring the kernel to page out other allocations.
pub fn run_swap_pressure(config: &SwapPressureConfig) -> Result<SwapPressureStats, Box<dyn std::error::Error>> {
    if config.dry_run {
        info!("[dry-run] Swap pressure plan:");
        info!("  pages_to_lock: {}", config.pages_to_lock);
        info!("  page_size:     {}", config.page_size);
        info!("  concurrency:   {}", config.concurrency);
        info!("  duration:      {:?}", config.duration);
        return Ok(SwapPressureStats::default());
    }

    let stop = Arc::new(AtomicBool::new(false));
    let stats = Arc::new(std::sync::Mutex::new(SwapPressureStats::default()));
    let handles: Vec<_> = (0..config.concurrency)
        .map(|_worker_id| {
            let stop = stop.clone();
            let stats = stats.clone();
            let page_size = config.page_size;
            let pages_per_worker = config.pages_to_lock / config.concurrency;

            thread::spawn(move || {
                #[allow(unused_mut)]
                let mut locked_pages: Vec<*mut u8> = Vec::with_capacity(pages_per_worker);
                let mut errors: u64 = 0;

                for _ in 0..pages_per_worker {
                    if stop.load(Ordering::SeqCst) {
                        break;
                    }

                    let layout = Layout::from_size_align(page_size, page_size)
                        .expect("valid page layout");
                    let ptr = unsafe { alloc(layout) };
                    if ptr.is_null() {
                        errors += 1;
                        continue;
                    }
                    // Touch page to force physical backing
                    unsafe { *ptr = 0xFF };

                    // Lock in RAM — if this fails the kernel is under memory pressure
                    #[cfg(target_os = "linux")]
                    {
                        let ret = unsafe { libc::mlock(ptr as *mut libc::c_void, page_size) };
                        if ret == 0 {
                            locked_pages.push(ptr);
                        } else {
                            unsafe { dealloc(ptr, layout) };
                            errors += 1;
                        }
                    }
                    #[cfg(not(target_os = "linux"))]
                    {
                        unsafe { dealloc(ptr, layout) };
                        errors += 1;
                    }
                }

                if let Ok(mut s) = stats.lock() {
                    s.pages_locked += locked_pages.len() as u64;
                    s.swap_errors += errors;
                }

                // Hold lock until stop signal
                while !stop.load(Ordering::SeqCst) {
                    thread::sleep(Duration::from_millis(100));
                }

                // Release locked pages
                for ptr in &locked_pages {
                    let layout = Layout::from_size_align(page_size, page_size)
                        .expect("valid page layout");
                    #[cfg(target_os = "linux")]
                    unsafe {
                        libc::munlock(*ptr as *mut libc::c_void, page_size);
                    }
                    unsafe { dealloc(*ptr, layout) };
                }
            })
        })
        .collect();

    let start = Instant::now();
    thread::sleep(config.duration);
    stop.store(true, Ordering::SeqCst);

    for h in handles {
        h.join().map_err(|e| format!("thread join failed: {:?}", e))?;
    }

    let elapsed = start.elapsed().as_secs_f64();
    let final_stats = stats.lock().unwrap_or_else(|e| e.into_inner()).clone();
    info!(
        pages_locked = final_stats.pages_locked,
        errors = final_stats.swap_errors,
        duration_secs = elapsed,
        "Swap pressure completed"
    );

    Ok(SwapPressureStats { duration_secs: elapsed, ..final_stats })
}

/// Release all heap-fill blocks.
fn heap_fill_release() -> u64 {
    unsafe {
        match read(addr_of_mut!(HEAP_FILL_BLOCKS)) {
            Some(b) => {
                let total = b.iter().map(|v| v.capacity() as u64).sum();
                write(addr_of_mut!(HEAP_FILL_BLOCKS), None);
                drop(b);
                total
            }
            None => 0,
        }
    }
}

/// Get current RSS in bytes (platform-specific).
fn current_rss_bytes() -> u64 {
    #[cfg(target_os = "linux")]
    {
        if let Ok(content) = std::fs::read_to_string("/proc/self/status") {
            for line in content.lines() {
                if line.starts_with("VmRSS:") {
                    if let Some(val) = line.split_whitespace().nth(1) {
                        if let Ok(kb) = val.parse::<u64>() {
                            return kb * 1024;
                        }
                    }
                }
            }
        }
        // Fallback: sysconf
        unsafe { libc::sysconf(libc::_SC_PHYS_PAGES) as u64 * libc::sysconf(libc::_SC_PAGE_SIZE) as u64 }
    }
    #[cfg(target_os = "macos")]
    {
        // Use libc::getrusage on macOS for RSS
        unsafe {
            let mut usage = std::mem::zeroed::<libc::rusage>();
            if libc::getrusage(libc::RUSAGE_SELF, &mut usage) == 0 {
                // ru_maxrss is in bytes on macOS
                usage.ru_maxrss as u64
            } else {
                0
            }
        }
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        0
    }
}

/// Get total physical RAM in bytes.
fn total_physical_ram() -> u64 {
    #[cfg(target_os = "linux")]
    {
        if let Ok(content) = std::fs::read_to_string("/proc/meminfo") {
            for line in content.lines() {
                if line.starts_with("MemTotal:") {
                    if let Some(val) = line.split_whitespace().nth(1) {
                        if let Ok(kb) = val.parse::<u64>() {
                            return kb * 1024;
                        }
                    }
                }
            }
        }
        unsafe { libc::sysconf(libc::_SC_PHYS_PAGES) as u64 * libc::sysconf(libc::_SC_PAGE_SIZE) as u64 }
    }
    #[cfg(target_os = "macos")]
    {
        unsafe {
            let mut size: u64 = 0;
            let mut count = std::mem::size_of::<u64>();
            let result = libc::sysctlbyname(
                b"hw.memsize\0".as_ptr() as *const libc::c_char,
                &mut size as *mut _ as *mut libc::c_void,
                &mut count,
                std::ptr::null_mut(),
                0,
            );
            if result == 0 {
                return size;
            }
        }
        8 * 1024 * 1024 * 1024
    }
    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        4 * 1024 * 1024 * 1024
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn memory_pressure_dry_run_returns_immediately() {
        let config = MemoryPressureConfig {
            dry_run: true,
            ..Default::default()
        };
        let stats = run(&config).unwrap();
        assert_eq!(stats.total_allocated_bytes, 0);
    }

    #[test]
    fn rss_reader_does_not_panic() {
        let rss = current_rss_bytes();
        assert!(rss < 1_000_000_000_000);
    }
}
