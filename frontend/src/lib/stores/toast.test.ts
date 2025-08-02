import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toast } from './toast';
import type { Toast } from './toast';

// Define constants for test values
const DEFAULT_TOAST_DURATION = 5000;
const SHORT_DURATION = 1000;
const MEDIUM_DURATION = 2000;
const LONG_DURATION = 3000;
const TIMER_PRECISION = 1; // ms
const LARGE_TIME = 10000;
const RAPID_TOAST_COUNT = 10;
const RAPID_TOAST_DURATION = 100;

describe('toast store', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		// Clear any existing toasts
		const currentToasts = get(toast);
		currentToasts.forEach(t => toast.remove(t.id));
	});

	afterEach(() => {
		vi.runOnlyPendingTimers();
	});

	it('should start with empty toast array', () => {
		const toasts = get(toast);
		expect(toasts).toEqual([]);
	});

	it('should add success toast', () => {
		const id = toast.success('Success message');
		const toasts = get(toast);

		expect(toasts).toHaveLength(1);
		expect(toasts[0]).toMatchObject({
			id,
			message: 'Success message',
			type: 'success',
			duration: DEFAULT_TOAST_DURATION
		});
	});

	it('should add error toast', () => {
		const id = toast.error('Error message');
		const toasts = get(toast);

		expect(toasts).toHaveLength(1);
		expect(toasts[0]).toMatchObject({
			id,
			message: 'Error message',
			type: 'error',
			duration: DEFAULT_TOAST_DURATION
		});
	});

	it('should add info toast', () => {
		const id = toast.info('Info message');
		const toasts = get(toast);

		expect(toasts).toHaveLength(1);
		expect(toasts[0]).toMatchObject({
			id,
			message: 'Info message',
			type: 'info',
			duration: DEFAULT_TOAST_DURATION
		});
	});

	it('should add warning toast', () => {
		const id = toast.warning('Warning message');
		const toasts = get(toast);

		expect(toasts).toHaveLength(1);
		expect(toasts[0]).toMatchObject({
			id,
			message: 'Warning message',
			type: 'warning',
			duration: DEFAULT_TOAST_DURATION
		});
	});

	it('should support custom duration', () => {
		const id = toast.success('Custom duration', 3000);
		const toasts = get(toast);

		expect(toasts[0]).toMatchObject({
			duration: 3000
		});
	});

	it('should generate unique IDs for each toast', () => {
		const id1 = toast.success('Message 1');
		const id2 = toast.error('Message 2');
		const id3 = toast.info('Message 3');

		expect(id1).not.toBe(id2);
		expect(id2).not.toBe(id3);
		expect(id1).not.toBe(id3);

		const toasts = get(toast);
		expect(toasts).toHaveLength(3);
		expect(new Set(toasts.map(t => t.id)).size).toBe(3);
	});

	it('should add multiple toasts', () => {
		toast.success('Message 1');
		toast.error('Message 2');
		toast.warning('Message 3');

		const toasts = get(toast);
		expect(toasts).toHaveLength(3);
		expect(toasts[0].message).toBe('Message 1');
		expect(toasts[1].message).toBe('Message 2');
		expect(toasts[2].message).toBe('Message 3');
	});

	it('should remove toast manually', () => {
		const id1 = toast.success('Message 1');
		const id2 = toast.error('Message 2');

		let toasts = get(toast);
		expect(toasts).toHaveLength(2);

		toast.remove(id1);

		toasts = get(toast);
		expect(toasts).toHaveLength(1);
		expect(toasts[0].id).toBe(id2);
	});

	it('should auto-remove toast after duration', () => {
		toast.success('Auto-remove', 1000);

		let toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Fast-forward time by 999ms - toast should still be there
		vi.advanceTimersByTime(999);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Fast-forward by 1ms more - toast should be removed
		vi.advanceTimersByTime(1);
		toasts = get(toast);
		expect(toasts).toHaveLength(0);
	});

	it('should handle zero duration (no auto-removal)', () => {
		toast.success('No auto-remove', 0);

		let toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Fast-forward significant time - toast should still be there
		vi.advanceTimersByTime(LARGE_TIME);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
	});

	it('should handle negative duration (no auto-removal)', () => {
		toast.success('No auto-remove', -1);

		let toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Fast-forward significant time - toast should still be there
		vi.advanceTimersByTime(LARGE_TIME);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
	});

	it('should remove multiple toasts independently', () => {
		const id1 = toast.success('Message 1', SHORT_DURATION);
		const id2 = toast.error('Message 2', MEDIUM_DURATION);
		const id3 = toast.info('Message 3', LONG_DURATION);

		let toasts = get(toast);
		expect(toasts).toHaveLength(3);

		// After SHORT_DURATION, first toast should be removed
		vi.advanceTimersByTime(SHORT_DURATION);
		toasts = get(toast);
		expect(toasts).toHaveLength(2);
		expect(toasts.map(t => t.id)).toEqual([id2, id3]);

		// After another SHORT_DURATION (MEDIUM_DURATION total), second toast should be removed
		vi.advanceTimersByTime(SHORT_DURATION);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
		expect(toasts[0].id).toBe(id3);

		// After another SHORT_DURATION (LONG_DURATION total), third toast should be removed
		vi.advanceTimersByTime(SHORT_DURATION);
		toasts = get(toast);
		expect(toasts).toHaveLength(0);
	});

	it('should handle removing non-existent toast gracefully', () => {
		toast.success('Message 1');

		let toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Try to remove non-existent toast
		toast.remove('non-existent-id');

		toasts = get(toast);
		expect(toasts).toHaveLength(1); // Should still have the original toast
	});

	it('should maintain toast order', () => {
		const id1 = toast.success('First');
		const id2 = toast.error('Second');
		const id3 = toast.info('Third');

		const toasts = get(toast);
		expect(toasts.map(t => t.id)).toEqual([id1, id2, id3]);
		expect(toasts.map(t => t.message)).toEqual(['First', 'Second', 'Third']);
	});

	it('should support store subscription', () => {
		const mockCallback = vi.fn();
		const unsubscribe = toast.subscribe(mockCallback);

		// Initial call with empty array
		expect(mockCallback).toHaveBeenCalledWith([]);

		// Add a toast
		toast.success('Test message');
		expect(mockCallback).toHaveBeenCalledTimes(2);

		const lastCall = mockCallback.mock.calls[1][0];
		expect(lastCall).toHaveLength(1);
		expect(lastCall[0].message).toBe('Test message');

		unsubscribe();
	});

	it('should handle rapid toast additions and removals', () => {
		// Add many toasts rapidly
		const ids: string[] = [];
		for (let i = 0; i < RAPID_TOAST_COUNT; i++) {
			ids.push(toast.success(`Message ${i}`, RAPID_TOAST_DURATION));
		}

		let toasts = get(toast);
		expect(toasts).toHaveLength(RAPID_TOAST_COUNT);

		// Remove some manually
		toast.remove(ids[0]);
		toast.remove(ids[5]);

		toasts = get(toast);
		expect(toasts).toHaveLength(RAPID_TOAST_COUNT - 2);

		// Let auto-removal kick in
		vi.advanceTimersByTime(RAPID_TOAST_DURATION);

		toasts = get(toast);
		expect(toasts).toHaveLength(0);
	});

	it('should return the correct toast ID format', () => {
		const id1 = toast.success('Test');
		const id2 = toast.error('Test');

		expect(id1).toMatch(/^toast-\d+$/);
		expect(id2).toMatch(/^toast-\d+$/);

		// IDs should be sequential
		const num1 = parseInt(id1.split('-')[1]);
		const num2 = parseInt(id2.split('-')[1]);
		expect(num2).toBe(num1 + 1);
	});

	it('should handle concurrent modifications during auto-removal', () => {
		// Add toast with short duration
		const id1 = toast.success('Auto-remove', RAPID_TOAST_DURATION);

		// Advance time to just before removal
		vi.advanceTimersByTime(RAPID_TOAST_DURATION - TIMER_PRECISION);

		// Add another toast while the first is about to be removed
		const id2 = toast.error('New toast', SHORT_DURATION);

		// Complete the first toast's timer
		vi.advanceTimersByTime(TIMER_PRECISION);

		// First toast should be removed, second should remain
		const toasts = get(toast);
		expect(toasts).toHaveLength(1);
		expect(toasts[0].id).toBe(id2);
	});

	it('should handle maximum number of toasts gracefully', () => {
		const MAX_TOASTS = 100;
		const ids: string[] = [];

		// Add maximum number of toasts
		for (let i = 0; i < MAX_TOASTS; i++) {
			ids.push(toast.success(`Toast ${i}`, 0)); // No auto-removal
		}

		const toasts = get(toast);
		expect(toasts).toHaveLength(MAX_TOASTS);

		// Verify all toasts have unique IDs
		const uniqueIds = new Set(toasts.map(t => t.id));
		expect(uniqueIds.size).toBe(MAX_TOASTS);

		// Clean up - remove all toasts
		ids.forEach(id => toast.remove(id));
		expect(get(toast)).toHaveLength(0);
	});

	it('should perform well with large numbers of toasts', () => {
		const LARGE_COUNT = 1000;
		const startTime = performance.now();

		// Add many toasts
		for (let i = 0; i < LARGE_COUNT; i++) {
			toast.info(`Performance test ${i}`, RAPID_TOAST_DURATION);
		}

		const addTime = performance.now() - startTime;
		expect(addTime).toBeLessThan(1000); // Should complete in under 1 second

		// Verify all were added
		expect(get(toast)).toHaveLength(LARGE_COUNT);

		// Measure auto-removal performance
		const removeStartTime = performance.now();
		vi.advanceTimersByTime(RAPID_TOAST_DURATION);
		const removeTime = performance.now() - removeStartTime;

		expect(removeTime).toBeLessThan(1000); // Should complete in under 1 second
		expect(get(toast)).toHaveLength(0);
	});

	it('should handle edge case of updating store during iteration', () => {
		// Set up subscription before adding toasts
		let modificationDone = false;
		const unsubscribe = toast.subscribe((toasts) => {
			// Only modify once when we have exactly 3 toasts
			if (!modificationDone && toasts.length === 3) {
				modificationDone = true;
				// Remove the middle toast during subscription callback
				const middleToast = toasts[1];
				toast.remove(middleToast.id);
			}
		});

		// Add initial toasts with 0 duration to prevent auto-removal
		const id1 = toast.success('First', 0);
		const id2 = toast.error('Second', 0);
		const id3 = toast.info('Third', 0);

		// Allow subscription callbacks to complete
		vi.advanceTimersByTime(0);

		// Verify the modification was handled correctly
		const finalToasts = get(toast);
		expect(finalToasts).toHaveLength(2);
		expect(finalToasts.map(t => t.message)).toEqual(['First', 'Third']);

		unsubscribe();
	});
});

// Note: The implementation should consider limiting the maximum number of toasts
// displayed to prevent UI overflow and performance issues
