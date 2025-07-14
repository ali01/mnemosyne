import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { debounce } from './debounce';

describe('debounce utility', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.runOnlyPendingTimers();
	});

	it('should call the function after the specified delay', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		debouncedFn('test');

		// Function should not be called immediately
		expect(mockFn).not.toHaveBeenCalled();

		// Fast-forward time by 100ms
		vi.advanceTimersByTime(100);

		// Function should now be called
		expect(mockFn).toHaveBeenCalledWith('test');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should only call the function once when called multiple times rapidly', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		// Call function multiple times rapidly
		debouncedFn('first');
		debouncedFn('second');
		debouncedFn('third');

		// Function should not be called yet
		expect(mockFn).not.toHaveBeenCalled();

		// Fast-forward time
		vi.advanceTimersByTime(100);

		// Function should be called only once with the last arguments
		expect(mockFn).toHaveBeenCalledWith('third');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should reset the timer when called again before delay expires', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		debouncedFn('first');

		// Advance time partially
		vi.advanceTimersByTime(50);

		// Call again before timer expires
		debouncedFn('second');

		// Advance time by another 50ms (total 100ms from start, but only 50ms from second call)
		vi.advanceTimersByTime(50);

		// Function should not be called yet
		expect(mockFn).not.toHaveBeenCalled();

		// Advance another 50ms to complete the debounce delay for second call
		vi.advanceTimersByTime(50);

		// Function should now be called with second arguments
		expect(mockFn).toHaveBeenCalledWith('second');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should work with zero delay', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 0);

		debouncedFn('test');

		// With zero delay, function should be called on next tick
		vi.advanceTimersByTime(0);

		expect(mockFn).toHaveBeenCalledWith('test');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should handle functions with multiple arguments', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		debouncedFn('arg1', 'arg2', { key: 'value' });

		vi.advanceTimersByTime(100);

		expect(mockFn).toHaveBeenCalledWith('arg1', 'arg2', { key: 'value' });
	});

	it('should handle functions with no arguments', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		debouncedFn();

		vi.advanceTimersByTime(100);

		expect(mockFn).toHaveBeenCalledWith();
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should maintain context when called as method', () => {
		const obj = {
			value: 'test',
			method: function(this: any, arg?: string) {
				return this.value + (arg || '');
			}
		};

		const methodSpy = vi.spyOn(obj, 'method');
		const debouncedMethod = debounce(obj.method.bind(obj), 100);
		debouncedMethod(' suffix');

		vi.advanceTimersByTime(100);

		expect(methodSpy).toHaveBeenCalledWith(' suffix');
		expect(methodSpy).toHaveBeenCalledTimes(1);
	});

	it('should handle rapid successive calls correctly', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, 100);

		// Simulate rapid calls over time
		for (let i = 0; i < 10; i++) {
			debouncedFn(`call-${i}`);
			vi.advanceTimersByTime(10); // Advance by 10ms each time
		}

		// Total time advanced: 100ms, but last call was only 10ms ago
		expect(mockFn).not.toHaveBeenCalled();

		// Advance by remaining time for last call
		vi.advanceTimersByTime(90);

		expect(mockFn).toHaveBeenCalledWith('call-9');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should allow separate debounced functions to work independently', () => {
		const mockFn1 = vi.fn();
		const mockFn2 = vi.fn();
		const debouncedFn1 = debounce(mockFn1, 100);
		const debouncedFn2 = debounce(mockFn2, 200);

		debouncedFn1('fn1');
		debouncedFn2('fn2');

		// After 100ms, first function should be called
		vi.advanceTimersByTime(100);
		expect(mockFn1).toHaveBeenCalledWith('fn1');
		expect(mockFn2).not.toHaveBeenCalled();

		// After another 100ms (200ms total), second function should be called
		vi.advanceTimersByTime(100);
		expect(mockFn2).toHaveBeenCalledWith('fn2');
	});
});
