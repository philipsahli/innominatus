'use client';

import { useEffect, useState, useRef } from 'react';
import { Send, Bot, User, AlertCircle, Loader2, Copy, Check } from 'lucide-react';
import { api, type ConversationMessage } from '@/lib/api';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import 'highlight.js/styles/github-dark.css';

export default function AssistantPage() {
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [aiAvailable, setAiAvailable] = useState<boolean | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Check AI status on mount
  useEffect(() => {
    async function checkStatus() {
      const response = await api.getAIStatus();
      setAiAvailable(response.success && response.data?.enabled ? true : false);
    }
    checkStatus();
  }, []);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || loading) return;

    const userMessage: ConversationMessage = {
      role: 'user',
      content: input.trim(),
      timestamp: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setLoading(true);

    try {
      const response = await api.sendAIChat(input.trim(), messages);

      if (response.success && response.data) {
        const assistantMessage: ConversationMessage = {
          role: 'assistant',
          content: response.data.message,
          timestamp: new Date().toISOString(),
          citations: response.data.citations,
        };
        setMessages((prev) => [...prev, assistantMessage]);
      } else {
        const errorMessage: ConversationMessage = {
          role: 'assistant',
          content: `Error: ${response.error || 'Failed to get response from AI assistant'}`,
          timestamp: new Date().toISOString(),
        };
        setMessages((prev) => [...prev, errorMessage]);
      }
    } catch (error) {
      const errorMessage: ConversationMessage = {
        role: 'assistant',
        content: `Error: ${error instanceof Error ? error.message : 'Unknown error occurred'}`,
        timestamp: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, errorMessage]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (aiAvailable === null) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-zinc-400" />
      </div>
    );
  }

  if (aiAvailable === false) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">AI Assistant</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Chat with the AI about innominatus platform
          </p>
        </div>

        <div className="flex flex-col items-center justify-center rounded-lg border border-amber-200 bg-amber-50 px-8 py-12 text-center dark:border-amber-900 dark:bg-amber-950">
          <AlertCircle className="h-10 w-10 text-amber-600 dark:text-amber-400" />
          <h2 className="mt-4 text-lg font-medium text-amber-900 dark:text-amber-100">
            AI Assistant Not Available
          </h2>
          <p className="mt-2 max-w-md text-sm text-amber-700 dark:text-amber-300">
            The AI assistant is not currently configured. Please set the required environment
            variables (ANTHROPIC_API_KEY) and restart the server.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">AI Assistant</h1>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Chat with the AI about innominatus platform, workflows, and resources
        </p>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto rounded-lg border border-zinc-200 bg-zinc-50 p-6 dark:border-zinc-800 dark:bg-zinc-900">
        {messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <Bot className="h-12 w-12 text-zinc-300 dark:text-zinc-700" />
            <p className="mt-4 text-sm text-zinc-500 dark:text-zinc-400">
              No messages yet. Start a conversation!
            </p>
            <div className="mt-6 grid gap-2 text-left">
              <p className="text-xs text-zinc-500 dark:text-zinc-400">Try asking:</p>
              <button
                onClick={() => setInput('How do I deploy an application?')}
                className="rounded border border-zinc-200 bg-white px-3 py-2 text-left text-sm text-zinc-700 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-300 dark:hover:bg-zinc-900"
              >
                How do I deploy an application?
              </button>
              <button
                onClick={() => setInput('What resource types are available?')}
                className="rounded border border-zinc-200 bg-white px-3 py-2 text-left text-sm text-zinc-700 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-300 dark:hover:bg-zinc-900"
              >
                What resource types are available?
              </button>
              <button
                onClick={() => setInput('Explain the provider architecture')}
                className="rounded border border-zinc-200 bg-white px-3 py-2 text-left text-sm text-zinc-700 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-300 dark:hover:bg-zinc-900"
              >
                Explain the provider architecture
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {messages.map((msg, idx) => (
              <Message key={idx} message={msg} />
            ))}
            {loading && (
              <div className="flex items-center gap-3 text-sm text-zinc-500">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>AI is thinking...</span>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Input */}
      <div className="mt-4 flex gap-2">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about innominatus..."
          disabled={loading}
          className="flex-1 resize-none rounded-lg border border-zinc-300 bg-white px-4 py-3 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white dark:focus:border-lime-400 dark:focus:ring-lime-400"
          rows={3}
        />
        <button
          onClick={handleSend}
          disabled={loading || !input.trim()}
          className="flex h-[88px] w-[88px] items-center justify-center rounded-lg bg-lime-600 text-white hover:bg-lime-700 disabled:opacity-50 disabled:hover:bg-lime-600 dark:bg-lime-500 dark:hover:bg-lime-600"
        >
          <Send className="h-5 w-5" />
        </button>
      </div>

      <p className="mt-2 text-xs text-zinc-500">Press Enter to send, Shift+Enter for new line</p>
    </div>
  );
}

function Message({ message }: { message: ConversationMessage }) {
  const isUser = message.role === 'user';
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  const handleCopyCode = (code: string, blockId: string) => {
    navigator.clipboard.writeText(code);
    setCopiedCode(blockId);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  return (
    <div className={`flex gap-3 ${isUser ? 'justify-end' : 'justify-start'}`}>
      {!isUser && (
        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-lime-500 dark:bg-lime-600">
          <Bot className="h-4 w-4 text-white" />
        </div>
      )}

      <div className={`max-w-[80%] ${isUser ? 'items-end' : 'items-start'} flex flex-col gap-1`}>
        <div
          className={`rounded-lg px-4 py-2 text-sm ${
            isUser
              ? 'bg-lime-500 text-white dark:bg-lime-600'
              : 'bg-white text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100'
          }`}
        >
          {isUser ? (
            <div className="whitespace-pre-wrap">{message.content}</div>
          ) : (
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
              components={{
                p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                ul: ({ children }) => (
                  <ul className="list-disc list-inside mb-2 space-y-1">{children}</ul>
                ),
                ol: ({ children }) => (
                  <ol className="list-decimal list-inside mb-2 space-y-1">{children}</ol>
                ),
                li: ({ children }) => <li className="ml-2">{children}</li>,
                h1: ({ children }) => <h1 className="text-lg font-bold mb-2 mt-2">{children}</h1>,
                h2: ({ children }) => <h2 className="text-base font-bold mb-2 mt-2">{children}</h2>,
                h3: ({ children }) => <h3 className="text-sm font-bold mb-1 mt-1">{children}</h3>,
                code: ({ inline, className, children, ...props }: any) => {
                  const codeString = String(children).replace(/\n$/, '');
                  const language = className?.replace(/language-/, '') || '';
                  const blockId = `code-${Math.random().toString(36).substr(2, 9)}`;

                  if (inline) {
                    return (
                      <code
                        className="bg-zinc-100 dark:bg-zinc-700 px-1.5 py-0.5 rounded text-xs font-mono"
                        {...props}
                      >
                        {children}
                      </code>
                    );
                  }

                  return (
                    <div className="relative group my-2">
                      {language && (
                        <div className="flex items-center justify-between bg-zinc-800 dark:bg-black px-3 py-1 rounded-t text-xs">
                          <span className="text-zinc-400 font-mono">{language}</span>
                          <button
                            onClick={() => handleCopyCode(codeString, blockId)}
                            className="text-zinc-400 hover:text-zinc-200 transition-colors"
                            title="Copy code"
                          >
                            {copiedCode === blockId ? (
                              <Check className="w-3 h-3" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                      )}
                      <code
                        className={`block bg-zinc-900 dark:bg-black text-zinc-100 p-3 text-xs font-mono overflow-x-auto ${
                          language ? 'rounded-b' : 'rounded'
                        }`}
                        {...props}
                      >
                        {children}
                      </code>
                    </div>
                  );
                },
                pre: ({ children }) => <>{children}</>,
                blockquote: ({ children }) => (
                  <blockquote className="border-l-4 border-zinc-300 dark:border-zinc-600 pl-3 italic my-2">
                    {children}
                  </blockquote>
                ),
                a: ({ children, href }) => (
                  <a
                    href={href}
                    className="text-lime-600 dark:text-lime-400 underline hover:no-underline"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {children}
                  </a>
                ),
                strong: ({ children }) => <strong className="font-bold">{children}</strong>,
                em: ({ children }) => <em className="italic">{children}</em>,
              }}
            >
              {message.content}
            </ReactMarkdown>
          )}
        </div>

        {message.citations && message.citations.length > 0 && (
          <div className="px-2 text-xs text-zinc-500">
            <p className="mb-1">Sources:</p>
            <ul className="space-y-0.5">
              {message.citations.map((citation, idx) => (
                <li key={idx} className="truncate">
                  • {citation}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      {isUser && (
        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-zinc-500 dark:bg-zinc-600">
          <User className="h-4 w-4 text-white" />
        </div>
      )}
    </div>
  );
}
