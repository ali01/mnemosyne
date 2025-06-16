export interface Node {
	id: string;
	title: string;
	position: {
		x: number;
		y: number;
		z?: number;
	};
	clusterId?: string;
	level: number;
	metadata?: Record<string, any>;
}

export interface Edge {
	id: string;
	source: string;
	target: string;
	weight: number;
	type: string;
}

export interface Cluster {
	id: string;
	level: number;
	centerNode: string;
	nodeCount: number;
	position: {
		x: number;
		y: number;
		z?: number;
	};
	radius: number;
}