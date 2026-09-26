// A simple foundation for central API client using standard fetch
const API_BASE = (window.location.port === '5173' ? 'http://localhost:3000/api/v1' : '/api/v1');

interface RequestOptions extends RequestInit {
  data?: any;
}

export const api = {
  async request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const url = `${API_BASE}${endpoint}`;
    
    const headers = new Headers(options.headers);
    if (!headers.has('Content-Type') && options.data && !(options.data instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
    }

    const match = document.cookie.match(new RegExp('(^| )csrf_=([^;]+)'));
    if (match) {
      headers.set('X-Csrf-Token', match[2]);
    }

    const config: RequestInit = {
      ...options,
      headers,
      credentials: 'include',
    };

    if (options.data) {
      config.body = options.data instanceof FormData ? options.data : JSON.stringify(options.data);
    }

    try {
      const response = await fetch(url, config);
      const isJson = response.headers.get('content-type')?.includes('application/json');
      
      const data = isJson ? await response.json() : await response.text();

      if (!response.ok) {
        throw new Error(data.message || 'API request failed');
      }

      return data as T;
    } catch (error) {
      console.error(`API Error [${endpoint}]:`, error);
      throw error;
    }
  },

  get<T>(endpoint: string, options?: RequestOptions) {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  },
  
  post<T>(endpoint: string, data: any, options?: RequestOptions) {
    return this.request<T>(endpoint, { ...options, method: 'POST', data });
  },

  put<T>(endpoint: string, data: any, options?: RequestOptions) {
    return this.request<T>(endpoint, { ...options, method: 'PUT', data });
  },

  delete<T>(endpoint: string, options?: RequestOptions) {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }
};
