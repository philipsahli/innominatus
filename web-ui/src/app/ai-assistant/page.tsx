'use client';

import React, { useState, useEffect, useRef } from 'react';
import { ProtectedRoute } from '@/components/protected-route';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ChatMessage } from '@/components/ai/chat-message';
import { ChatInput } from '@/components/ai/chat-input';
import { SpecPreview } from '@/components/ai/spec-preview';
import { Alert } from '@/components/ui/alert';
import { api, AIChatResponse, AIGenerateSpecResponse, ConversationMessage } from '@/lib/api';
import { Bot, Sparkles, AlertCircle, RefreshCw, Trash2 } from 'lucide-react';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  citations?: string[];
  spec?: string; // Store generated spec with the message
}

interface GeneratedSpec {
  spec: string;
  explanation: string;
  citations?: string[];
}

export default function AIAssistantPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [aiStatus, setAIStatus] = useState<{
    enabled: boolean;
    message?: string;
    llm_provider?: string;
    documents_loaded?: number;
  } | null>(null);
  const [generatedSpec, setGeneratedSpec] = useState<GeneratedSpec | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Check AI service status on mount
    const checkStatus = async () => {
      const response = await api.getAIStatus();
      if (response.success && response.data) {
        setAIStatus(response.data);
        if (!response.data.enabled) {
          setError(response.data.message || 'AI service is not available');
        }
      } else {
        setError('Failed to connect to AI service');
      }
    };
    checkStatus();
  }, []);

  useEffect(() => {
    // Auto-scroll to bottom when messages change
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSendMessage = async (message: string) => {
    if (!aiStatus?.enabled) {
      setError('AI service is not enabled');
      return;
    }

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: message,
      timestamp: new Date().toLocaleTimeString(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setLoading(true);
    setError(null);

    try {
      // Build conversation history from existing messages
      const conversationHistory: ConversationMessage[] = messages.map((msg) => ({
        role: msg.role,
        content: msg.content,
        timestamp: msg.timestamp,
        spec: msg.spec, // Include generated spec in history
      }));

      const response = await api.sendAIChat(message, conversationHistory);
      if (response.success && response.data) {
        const assistantMessage: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: response.data.message,
          timestamp: new Date(response.data.timestamp).toLocaleTimeString(),
          citations: response.data.citations,
          spec: response.data.generated_spec, // Store spec with message
        };

        setMessages((prev) => [...prev, assistantMessage]);

        // Check if spec was generated
        if (response.data.generated_spec) {
          setGeneratedSpec({
            spec: response.data.generated_spec,
            explanation: response.data.message,
            citations: response.data.citations,
          });
        }
      } else {
        setError(response.error || 'Failed to get AI response');
      }
    } catch (err) {
      setError('An error occurred while sending your message');
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateSpec = async (description: string) => {
    if (!aiStatus?.enabled) {
      setError('AI service is not enabled');
      return;
    }

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: `Generate a Score specification: ${description}`,
      timestamp: new Date().toLocaleTimeString(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setLoading(true);
    setError(null);

    try {
      const response = await api.generateSpec(description);
      if (response.success && response.data) {
        const assistantMessage: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: response.data.explanation,
          timestamp: new Date().toLocaleTimeString(),
          citations: response.data.citations,
          spec: response.data.spec, // IMPORTANT: Store spec in message for conversation history!
        };

        setMessages((prev) => [...prev, assistantMessage]);
        setGeneratedSpec({
          spec: response.data.spec,
          explanation: response.data.explanation,
          citations: response.data.citations,
        });
      } else {
        setError(response.error || 'Failed to generate spec');
      }
    } catch (err) {
      setError('An error occurred while generating spec');
    } finally {
      setLoading(false);
    }
  };

  const handleDeploy = async (spec: string) => {
    // TODO: Implement deployment via API
    alert('Deployment feature coming soon! For now, download the spec and deploy manually.');
  };

  const handleClearChat = () => {
    if (confirm('Are you sure you want to clear the chat history?')) {
      setMessages([]);
      setGeneratedSpec(null);
    }
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900">
        <div className="p-6 space-y-6">
          {/* Header */}
          <div className="relative">
            <div className="p-8 rounded-2xl border bg-white dark:bg-gray-900">
              <div className="flex items-center justify-between">
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900">
                      <Bot className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                    </div>
                    <h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100">
                      AI Assistant
                    </h1>
                  </div>
                  <p className="text-lg text-muted-foreground max-w-2xl">
                    Ask questions about innominatus, get help with workflows, and generate Score
                    specifications
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  {aiStatus && (
                    <div
                      className={`hidden md:flex items-center gap-2 px-4 py-2 rounded-lg ${aiStatus.enabled ? 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-400' : 'bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-400'}`}
                    >
                      <div
                        className={`w-2 h-2 rounded-full ${aiStatus.enabled ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}
                      ></div>
                      <span className="text-sm font-medium">
                        {aiStatus.enabled
                          ? `AI Ready (${aiStatus.documents_loaded} docs)`
                          : 'AI Unavailable'}
                      </span>
                    </div>
                  )}
                  <Button
                    onClick={handleClearChat}
                    variant="outline"
                    size="sm"
                    disabled={messages.length === 0}
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    Clear
                  </Button>
                </div>
              </div>
            </div>
          </div>

          {/* Error Display */}
          {error && (
            <Alert className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20">
              <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
              <div className="ml-2 text-sm text-red-800 dark:text-red-400">{error}</div>
            </Alert>
          )}

          {/* Main Content */}
          <div className="grid gap-6 lg:grid-cols-3">
            {/* Chat Area */}
            <Card className="lg:col-span-2 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg flex flex-col h-[600px]">
              <CardHeader className="pb-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900">
                      <Sparkles className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                    </div>
                    <CardTitle className="text-xl">Chat</CardTitle>
                  </div>
                  {loading && (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      Thinking...
                    </div>
                  )}
                </div>
              </CardHeader>
              <CardContent className="flex-1 overflow-y-auto p-4">
                {messages.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-center">
                    <div className="space-y-4 max-w-md">
                      <Bot className="w-16 h-16 text-muted-foreground mx-auto" />
                      <div>
                        <p className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                          Welcome to the AI Assistant!
                        </p>
                        <p className="text-sm text-muted-foreground">I can help you with:</p>
                        <ul className="text-sm text-muted-foreground mt-2 space-y-1">
                          <li>• Understanding innominatus features and workflows</li>
                          <li>• Generating Score specifications</li>
                          <li>• Answering questions about deployment and configuration</li>
                          <li>• Explaining golden paths and best practices</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                ) : (
                  <>
                    {messages.map((msg) => (
                      <ChatMessage
                        key={msg.id}
                        role={msg.role}
                        content={msg.content}
                        timestamp={msg.timestamp}
                        citations={msg.citations}
                      />
                    ))}
                    <div ref={messagesEndRef} />
                  </>
                )}
              </CardContent>
              <ChatInput
                onSend={handleSendMessage}
                onGenerateSpec={handleGenerateSpec}
                disabled={loading || !aiStatus?.enabled}
              />
            </Card>

            {/* Sidebar */}
            <div className="space-y-6">
              {/* Generated Spec */}
              {generatedSpec && (
                <SpecPreview
                  spec={generatedSpec.spec}
                  explanation={generatedSpec.explanation}
                  citations={generatedSpec.citations}
                  onDeploy={handleDeploy}
                />
              )}

              {/* Quick Actions */}
              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Quick Actions</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => handleSendMessage('What golden paths are available?')}
                    disabled={loading || !aiStatus?.enabled}
                  >
                    View Golden Paths
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => handleSendMessage('How do I deploy a microservice?')}
                    disabled={loading || !aiStatus?.enabled}
                  >
                    Deploy Microservice
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() =>
                      handleGenerateSpec('Node.js web application with PostgreSQL database')
                    }
                    disabled={loading || !aiStatus?.enabled}
                  >
                    Generate Example Spec
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => handleSendMessage('Explain the demo environment setup')}
                    disabled={loading || !aiStatus?.enabled}
                  >
                    Demo Environment Help
                  </Button>
                </CardContent>
              </Card>

              {/* Tips */}
              <Card className="bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base text-blue-900 dark:text-blue-400">Tips</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="text-sm text-blue-800 dark:text-blue-300 space-y-2">
                    <li>• Use the ✨ button to generate Score specifications</li>
                    <li>• Ask about specific workflows or golden paths</li>
                    <li>• Request examples for different application types</li>
                    <li>• All responses include source citations</li>
                  </ul>
                </CardContent>
              </Card>

              {/* Knowledge Base Resources */}
              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Knowledge Base</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-muted-foreground mb-3">
                    The AI assistant has access to the following innominatus resources:
                  </p>
                  <div className="space-y-3">
                    <div>
                      <p className="text-xs font-semibold text-gray-900 dark:text-gray-100 mb-1">
                        📚 Documentation
                      </p>
                      <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                        <li>• Architecture & Design</li>
                        <li>• User Guide</li>
                        <li>• CLI Reference</li>
                        <li>• API Documentation</li>
                        <li>• OIDC Authentication</li>
                        <li>• Health & Monitoring</li>
                        <li>• Observability Setup</li>
                      </ul>
                    </div>
                    <div>
                      <p className="text-xs font-semibold text-gray-900 dark:text-gray-100 mb-1">
                        ⚙️ Workflows
                      </p>
                      <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                        <li>• deploy-app</li>
                        <li>• undeploy-app</li>
                        <li>• ephemeral-env</li>
                        <li>• db-lifecycle</li>
                        <li>• observability-setup</li>
                      </ul>
                    </div>
                    <div>
                      <p className="text-xs font-semibold text-gray-900 dark:text-gray-100 mb-1">
                        📖 Additional
                      </p>
                      <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                        <li>• README.md</li>
                        <li>• CLAUDE.md (project guide)</li>
                        <li>• Golden paths config</li>
                        <li>• Score specifications</li>
                      </ul>
                    </div>
                    {aiStatus?.documents_loaded && (
                      <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                        <p className="text-xs text-muted-foreground">
                          <span className="font-semibold text-gray-900 dark:text-gray-100">
                            {aiStatus.documents_loaded}
                          </span>{' '}
                          documents loaded
                        </p>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </div>
    </ProtectedRoute>
  );
}
