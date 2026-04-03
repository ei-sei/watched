export interface ApiError {
  detail: string | { msg: string; type: string }[]
}

export interface MediaListParams {
  media_type?: string
  status?: string
  sort?: string
  order?: string
  page?: number
  per_page?: number
  q?: string
}
