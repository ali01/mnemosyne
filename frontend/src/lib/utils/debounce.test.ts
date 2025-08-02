import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { debounce } from './debounce';

// Define constants for test values
const DEFAULT_DELAY = 100;
const SHORT_DELAY = 10;
const LONG_DELAY = 200;
describe('debounce utility', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		// Proper timer cleanup to prevent test pollution
		vi.clearAllTimers();
		vi.useRealTimers();
	});

	it('should call the function after the specified delay', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		debouncedFn('test');

		// Function should not be called immediately
		expect(mockFn).not.toHaveBeenCalled();

		// Fast-forward time by 100ms
		vi.advanceTimersByTime(DEFAULT_DELAY);

		// Function should now be called
		expect(mockFn).toHaveBeenCalledWith('test');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should only call the function once when called multiple times rapidly', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		// Call function multiple times rapidly
		debouncedFn('first');
		debouncedFn('second');
		debouncedFn('third');

		// Function should not be called yet
		expect(mockFn).not.toHaveBeenCalled();

		// Fast-forward time
		vi.advanceTimersByTime(DEFAULT_DELAY);

		// Function should be called only once with the last arguments
		expect(mockFn).toHaveBeenCalledWith('third');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should reset the timer when called again before delay expires', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		debouncedFn('first');

		// Advance time partially
		vi.advanceTimersByTime(DEFAULT_DELAY / 2);

		// Call again before timer expires
		debouncedFn('second');

		// Advance time by another half delay (total DEFAULT_DELAY from start, but only half from second call)
		vi.advanceTimersByTime(DEFAULT_DELAY / 2);

		// Function should not be called yet
		expect(mockFn).not.toHaveBeenCalled();

		// Advance another half delay to complete the debounce delay for second call
		vi.advanceTimersByTime(DEFAULT_DELAY / 2);

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
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		debouncedFn('arg1', 'arg2', { key: 'value' });

		vi.advanceTimersByTime(DEFAULT_DELAY);

		expect(mockFn).toHaveBeenCalledWith('arg1', 'arg2', { key: 'value' });
	});

	it('should handle functions with no arguments', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		debouncedFn();

		vi.advanceTimersByTime(DEFAULT_DELAY);

		expect(mockFn).toHaveBeenCalledWith();
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should maintain context when called as method', () => {
		// Define proper types for context test
		type TestObject = {
			value: string;
			method: (arg?: string) => string;
		};

		const obj: TestObject = {
			value: 'test',
			method: function(arg?: string) {
				return this.value + (arg || '');
			}
		};

		// Test that return value is preserved
		const returnValues: string[] = [];
		const wrappedMethod = vi.fn(obj.method.bind(obj));
		const debouncedMethod = debounce((...args: Parameters<typeof wrappedMethod>) => {
			const result = wrappedMethod(...args);
			returnValues.push(result);
			return result;
		}, DEFAULT_DELAY);

		debouncedMethod(' suffix');

		vi.advanceTimersByTime(DEFAULT_DELAY);

		expect(wrappedMethod).toHaveBeenCalledWith(' suffix');
		expect(wrappedMethod).toHaveBeenCalledTimes(1);
		expect(returnValues).toEqual(['test suffix']);
	});

	it('should handle rapid successive calls correctly', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		// Simulate rapid calls over time
		for (let i = 0; i < 10; i++) {
			debouncedFn(`call-${i}`);
			vi.advanceTimersByTime(SHORT_DELAY); // Advance by SHORT_DELAY each time
		}

		// Total time advanced: 100ms, but last call was only SHORT_DELAY ago
		expect(mockFn).not.toHaveBeenCalled();

		// Advance by remaining time for last call
		vi.advanceTimersByTime(DEFAULT_DELAY - SHORT_DELAY);

		expect(mockFn).toHaveBeenCalledWith('call-9');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should allow separate debounced functions to work independently', () => {
		const mockFn1 = vi.fn();
		const mockFn2 = vi.fn();
		const debouncedFn1 = debounce(mockFn1, DEFAULT_DELAY);
		const debouncedFn2 = debounce(mockFn2, LONG_DELAY);

		debouncedFn1('fn1');
		debouncedFn2('fn2');

		// After 100ms, first function should be called
		vi.advanceTimersByTime(DEFAULT_DELAY);
		expect(mockFn1).toHaveBeenCalledWith('fn1');
		expect(mockFn2).not.toHaveBeenCalled();

		// After another 100ms (200ms total), second function should be called
		vi.advanceTimersByTime(DEFAULT_DELAY);
		expect(mockFn2).toHaveBeenCalledWith('fn2');
	});

	it('should handle errors thrown by the debounced function', () => {
		const errorMessage = 'Test error';
		const mockFn = vi.fn().mockImplementation(() => {
			throw new Error(errorMessage);
		});
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		// Expect the error to be thrown when the debounced function executes
		debouncedFn('test');

		// Advance time to trigger the debounced function
		expect(() => {
			vi.advanceTimersByTime(DEFAULT_DELAY);
		}).toThrow(errorMessage);

		// Verify the function was called
		expect(mockFn).toHaveBeenCalledWith('test');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should handle async errors in the debounced function', () => {
		const errorMessage = 'Async test error';
		const mockFn = vi.fn().mockRejectedValue(new Error(errorMessage));
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		debouncedFn('test');

		// Advance time to trigger the debounced function
		vi.advanceTimersByTime(DEFAULT_DELAY);

		// Verify the function was called
		expect(mockFn).toHaveBeenCalledWith('test');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});

	it('should not execute concurrently if execution time exceeds debounce delay', async () => {
		let executionCount = 0;
		let isExecuting = false;

		const slowFn = vi.fn(async () => {
			if (isExecuting) {
				throw new Error('Concurrent execution detected!');
			}
			isExecuting = true;
			executionCount++;

			// Simulate slow async operation
			await new Promise(resolve => setTimeout(resolve, LONG_DELAY));

			isExecuting = false;
		});

		const debouncedFn = debounce(slowFn, SHORT_DELAY);

		// Call the function multiple times rapidly
		debouncedFn();
		debouncedFn();
		debouncedFn();

		// Now advance time to trigger the debounced function
		vi.advanceTimersByTime(SHORT_DELAY);

		// Only one execution should happen
		expect(slowFn).toHaveBeenCalledTimes(1);
		expect(executionCount).toBe(1);

		// Wait for the slow function to complete
		await vi.advanceTimersByTimeAsync(LONG_DELAY);
	});

	it('should cancel previous timer when called again', () => {
		const mockFn = vi.fn();
		const debouncedFn = debounce(mockFn, DEFAULT_DELAY);

		// First call
		debouncedFn('first');

		// Advance time partially
		vi.advanceTimersByTime(DEFAULT_DELAY - 1);

		// Second call should cancel the first
		debouncedFn('second');

		// Advance just 1ms more (which would have triggered first call)
		vi.advanceTimersByTime(1);

		// First call should not have executed
		expect(mockFn).not.toHaveBeenCalled();

		// Advance to complete second call's delay
		vi.advanceTimersByTime(DEFAULT_DELAY - 1);

		// Only second call should execute
		expect(mockFn).toHaveBeenCalledWith('second');
		expect(mockFn).toHaveBeenCalledTimes(1);
	});
});
