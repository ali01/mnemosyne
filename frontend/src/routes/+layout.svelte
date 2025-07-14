<script lang="ts">
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	import ErrorBoundary from '$lib/components/ErrorBoundary.svelte';
</script>

<Toast />
<ErrorBoundary>
	<slot />
</ErrorBoundary>