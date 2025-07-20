import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toast } from './toast';
import type { Toast } from './toast';

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
			duration: 5000
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
			duration: 5000
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
			duration: 5000
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
			duration: 5000
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
		vi.advanceTimersByTime(10000);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
	});

	it('should handle negative duration (no auto-removal)', () => {
		toast.success('No auto-remove', -1);

		let toasts = get(toast);
		expect(toasts).toHaveLength(1);

		// Fast-forward significant time - toast should still be there
		vi.advanceTimersByTime(10000);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
	});

	it('should remove multiple toasts independently', () => {
		const id1 = toast.success('Message 1', 1000);
		const id2 = toast.error('Message 2', 2000);
		const id3 = toast.info('Message 3', 3000);

		let toasts = get(toast);
		expect(toasts).toHaveLength(3);

		// After 1000ms, first toast should be removed
		vi.advanceTimersByTime(1000);
		toasts = get(toast);
		expect(toasts).toHaveLength(2);
		expect(toasts.map(t => t.id)).toEqual([id2, id3]);

		// After another 1000ms (2000ms total), second toast should be removed
		vi.advanceTimersByTime(1000);
		toasts = get(toast);
		expect(toasts).toHaveLength(1);
		expect(toasts[0].id).toBe(id3);

		// After another 1000ms (3000ms total), third toast should be removed
		vi.advanceTimersByTime(1000);
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
		for (let i = 0; i < 10; i++) {
			ids.push(toast.success(`Message ${i}`, 100));
		}

		let toasts = get(toast);
		expect(toasts).toHaveLength(10);

		// Remove some manually
		toast.remove(ids[0]);
		toast.remove(ids[5]);

		toasts = get(toast);
		expect(toasts).toHaveLength(8);

		// Let auto-removal kick in
		vi.advanceTimersByTime(100);

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
});
