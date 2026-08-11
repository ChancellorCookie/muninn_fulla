const BASE = '';

async function api<T>(path: string, options?: RequestInit): Promise<T> {
	const url = `${BASE}/api${path}`;
	console.log('FETCH', options?.method || 'GET', url);
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), 10000);
	try {
		const res = await fetch(url, {
			headers: { 'Content-Type': 'application/json' },
			signal: controller.signal,
			...options,
		});
		clearTimeout(timer);
		if (!res.ok) {
			const body = await res.text();
			throw new Error(body || res.statusText);
		}
		return res.json();
	} catch(e: any) {
		clearTimeout(timer);
		console.error('FETCH ERROR', url, e);
		throw e;
	}
}

export const accounts = {
	list: () => api<import('./types').Account[]>('/accounts'),
	create: (data: { name: string; type: string }) =>
		api<import('./types').Account>('/accounts', { method: 'POST', body: JSON.stringify(data) }),
	del: (id: string) => api<void>(`/accounts/${id}`, { method: 'DELETE' }),
};

export const categories = {
	list: (income?: boolean) => {
		const q = income !== undefined ? `?income=${income}` : '';
		return api<import('./types').Category[]>(`/categories${q}`);
	},
	create: (data: { name: string; color: string; icon: string; is_income: boolean }) =>
		api<import('./types').Category>('/categories', { method: 'POST', body: JSON.stringify(data) }),
	del: (id: string) => api<void>(`/categories/${id}`, { method: 'DELETE' }),
	update: (id: string, data: { name: string; color: string }) =>
		api<void>(`/categories/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
};

export const transactions = {
	list: (filters?: { account?: string; month?: string; status?: string; search?: string }) => {
		const params = new URLSearchParams();
		if (filters?.account) params.set('account', filters.account);
		if (filters?.month) params.set('month', filters.month);
		if (filters?.status) params.set('status', filters.status);
		if (filters?.search) params.set('search', filters.search);
		const q = params.toString();
		return api<import('./types').Transaction[]>(`/transactions${q ? '?' + q : ''}`);
	},
	create: (data: { account_id: string; category_id: string; amount: number; description: string; date: string; note?: string }) =>
		api<import('./types').Transaction>('/transactions', { method: 'POST', body: JSON.stringify(data) }),
	del: (id: string) => api<void>(`/transactions/${id}`, { method: 'DELETE' }),
	update: (id: string, data: any) => api<void>(`/transactions/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
	toggleExclude: (id: string) => api<void>(`/transactions/toggle-exclude/${id}`, { method: 'POST' }),
	bulkUpdate: (data: { ids: string[]; category_id?: string; status?: string }) =>
		api<{ updated: number }>('/transactions/bulk', { method: 'PATCH', body: JSON.stringify(data) }),
};

export const summary = {
	month: (month: string, account?: string) => {
		const params = new URLSearchParams({ month });
		if (account) params.set('account', account);
		return api<import('./types').MonthSummary>(`/summary?${params}`);
	},
};

export const recurring = {
	list: () => api<import('./types').RecurringTransaction[]>('/recurring'),
	create: (data: { account_id: string; category_id: string; amount: number; description: string; interval_kind: string; interval_n?: number; next_due: string }) =>
		api<import('./types').RecurringTransaction>('/recurring', { method: 'POST', body: JSON.stringify(data) }),
	update: (id: string, data: { account_id: string; category_id: string; amount: number; description: string; interval_kind: string; interval_n?: number; next_due: string }) =>
		api<void>(`/recurring/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
	toggle: (id: string) => api<void>(`/recurring/${id}`, { method: 'POST' }),
	del: (id: string) => api<void>(`/recurring/${id}`, { method: 'DELETE' }),
	history: (id: string) => api<import('./types').Transaction[]>(`/recurring/${id}/history`),
	process: () => api<{ processed: number }>('/recurring/process', { method: 'POST' }),
};

export const importCsv = {
	upload: async (file: File, accountId: string): Promise<{ imported: number; skipped: number; auto_posted: number }> => {
		const form = new FormData();
		form.append('file', file);
		const res = await fetch(`${BASE}/api/import/csv?account=${accountId}`, {
			method: 'POST',
			body: form,
		});
		if (!res.ok) throw new Error(await res.text());
		return res.json();
	},
};
