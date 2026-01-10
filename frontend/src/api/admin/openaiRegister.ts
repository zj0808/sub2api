/**
 * Admin OpenAI Register API endpoints
 * Handles OpenAI account auto-registration and session-to-RT conversion
 */

import { apiClient } from '../client'

export interface AutoRegisterRequest {
  email: string
  password: string
  proxy_id?: number
  name?: string
  group_ids?: number[]
  priority?: number
  concurrency?: number
  create_account?: boolean
}

export interface AutoRegisterResponse {
  success: boolean
  email: string
  refresh_token?: string
  account_id?: number
  error?: string
}

export interface SessionToRTRequest {
  session_token: string
  proxy_id?: number
  name?: string
  group_ids?: number[]
  priority?: number
  concurrency?: number
  create_account?: boolean
}

export interface SessionToRTResponse {
  success: boolean
  refresh_token?: string
  account_id?: number
  error?: string
}

/**
 * Auto register OpenAI account and get RT
 * @param request - Registration parameters
 * @returns Registration result with RT
 */
export async function autoRegister(request: AutoRegisterRequest): Promise<AutoRegisterResponse> {
  const { data } = await apiClient.post<AutoRegisterResponse>('/admin/openai/auto-register', request)
  return data
}

/**
 * Convert session token to refresh token
 * @param request - Session token and options
 * @returns RT conversion result
 */
export async function sessionToRT(request: SessionToRTRequest): Promise<SessionToRTResponse> {
  const { data } = await apiClient.post<SessionToRTResponse>('/admin/openai/session-to-rt', request)
  return data
}

// 邮局相关接口
export interface FetchEmailCodeRequest {
  to_email: string
  admin_email: string
  admin_password: string
  base_url?: string
}

export interface FetchEmailCodeResponse {
  code: string
}

export interface CreateMailUserRequest {
  email: string
  domain: string
  password?: string
  admin_email: string
  admin_password: string
  base_url?: string
}

export interface CreateMailUserResponse {
  email: string
  password: string
}

/**
 * Fetch OpenAI verification code from email
 */
export async function fetchEmailCode(request: FetchEmailCodeRequest): Promise<FetchEmailCodeResponse> {
  const { data } = await apiClient.post<FetchEmailCodeResponse>('/admin/openai/fetch-email-code', request)
  return data
}

/**
 * Create a new mail user
 */
export async function createMailUser(request: CreateMailUserRequest): Promise<CreateMailUserResponse> {
  const { data } = await apiClient.post<CreateMailUserResponse>('/admin/openai/create-mail-user', request)
  return data
}

export const openaiRegisterAPI = {
  autoRegister,
  sessionToRT,
  fetchEmailCode,
  createMailUser
}

export default openaiRegisterAPI

