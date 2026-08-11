export interface Account {
	id: string;
	name: string;
	type: string;
	balance: number;
	currency: string;
	created_at: string;
	updated_at: string;
}

export interface Category {
	id: string;
	name: string;
	color: string;
	icon: string;
	is_income: boolean;
	created_at: string;
}

export interface Transaction {
	id: string;
	account_id: string;
	category_id: string;
	amount: number;
	description: string;
	date: string;
	note: string;
	status: string;
	created_at: string;
}

export interface RecurringTransaction {
	id: string;
	account_id: string;
	category_id: string;
	amount: number;
	description: string;
	interval_kind: string;
	interval_n: number;
	next_due: string;
	active: boolean;
	created_at: string;
}

export interface ForecastResponse {
	month: string;
	income: number;
	expenses: number;
	balance: number;
	by_cat: CategoryForecast[];
}
export interface CategoryForecast {
	category_id: string;
	category_name: string;
	color: string;
	total: number;
	items: { name: string; amount: number }[];
}

export interface CategorySummary {
	category_id: string;
	category_name: string;
	color: string;
	amount: number;
	count: number;
}

export interface HealthResponse {
	status: string;
	version: string;
	now: string;
}
