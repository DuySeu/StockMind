export type Document = {
  id: string;
  name: string;
  transaction_id: string;
  template_id: string;
  template_name: string;
  status: string;
  flow_id: string;
  quality: string;
  children?: Document[];
};

export type FileContent = {
  base64: string;
  type: string;
  size: number;
};

export type FileDetail = {
  raw_file: string;
  processed_result: Record<string, any>;
  quality_result: {
    page: number;
    quality_metrics: QualityMetrics;
  }[];
};

export type QualityMetrics = {
  background_noise: number;
  brightness: number;
  brightness_uniformity: number;
  contrast: number;
  edge_strength: number;
  focus_score: number;
  is_blurry: boolean;
  is_low_contrast: boolean;
  is_low_resolution: boolean;
  is_overexposed: boolean;
  is_skewed: boolean;
  is_underexposed: boolean;
  is_uneven_lighting: boolean;
  lighting_uniformity: number;
  overall_quality: number;
  resolution_score: number;
  skew_angle: number;
  text_background_separation: number;
  text_clarity: number;
  white_balance_score: number;
};
