export interface ChatOption {
  id: string;
  label: string;
}

export interface ChatResponse {
  type: 'menu' | 'answer';
  message: string;
  options: ChatOption[];
}

export interface ChatHistoryItem {
  id: number;
  role: 'rider' | 'bot';
  text: string;
  created_at: string;
}
