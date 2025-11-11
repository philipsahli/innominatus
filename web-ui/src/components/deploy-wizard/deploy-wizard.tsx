import { useState } from 'react';
import { X } from 'lucide-react';
import { WizardData } from './types';
import { StepBasicInfo } from './step-basic-info';
import { StepContainer } from './step-container';
import { StepResources } from './step-resources';
import { StepReview } from './step-review';
import { YamlPreview } from './yaml-preview';
import { api } from '@/lib/api';

interface DeployWizardProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function DeployWizard({ isOpen, onClose, onSuccess }: DeployWizardProps) {
  const [currentStep, setCurrentStep] = useState(1);
  const [wizardData, setWizardData] = useState<WizardData>({
    appName: '',
    environment: 'kubernetes',
    ttl: '24h',
    container: {
      image: '',
      port: 8080,
      envVars: {},
    },
    resources: [],
  });

  const updateWizardData = (updates: Partial<WizardData>) => {
    setWizardData((prev) => ({
      ...prev,
      ...updates,
    }));
  };

  const handleNext = () => {
    setCurrentStep((prev) => Math.min(prev + 1, 4));
  };

  const handlePrev = () => {
    setCurrentStep((prev) => Math.max(prev - 1, 1));
  };

  const handleSubmit = async (yaml: string) => {
    try {
      const response = await api.submitSpec(yaml);

      if (response.success) {
        // Close wizard and refresh applications list
        onClose();
        if (onSuccess) {
          onSuccess();
        }
      } else {
        throw new Error(response.error || 'Failed to deploy application');
      }
    } catch (error) {
      // Error will be caught and displayed by StepReview component
      throw error;
    }
  };

  const handleClose = () => {
    // Reset wizard state
    setCurrentStep(1);
    setWizardData({
      appName: '',
      environment: 'kubernetes',
      ttl: '24h',
      container: {
        image: '',
        port: 8080,
        envVars: {},
      },
      resources: [],
    });
    onClose();
  };

  if (!isOpen) {
    return null;
  }

  const steps = [
    { number: 1, label: 'Basic Info' },
    { number: 2, label: 'Container' },
    { number: 3, label: 'Resources' },
    { number: 4, label: 'Review' },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="relative w-full max-w-7xl max-h-[90vh] overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-950">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-200 px-6 py-4 dark:border-zinc-800">
          <div>
            <h2 className="text-xl font-semibold text-zinc-900 dark:text-white">
              Deploy New Application
            </h2>
            <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">Step {currentStep} of 4</p>
          </div>
          <button
            onClick={handleClose}
            className="rounded-lg p-2 hover:bg-zinc-100 dark:hover:bg-zinc-900"
          >
            <X size={20} className="text-zinc-600 dark:text-zinc-400" />
          </button>
        </div>

        {/* Progress Steps */}
        <div className="border-b border-zinc-200 bg-zinc-50 px-6 py-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="flex items-center justify-between">
            {steps.map((step, index) => (
              <div key={step.number} className="flex items-center">
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                    step.number < currentStep
                      ? 'bg-lime-500 text-white'
                      : step.number === currentStep
                        ? 'bg-lime-500 text-white'
                        : 'bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400'
                  }`}
                >
                  {step.number}
                </div>
                <span
                  className={`ml-2 text-sm font-medium ${
                    step.number === currentStep
                      ? 'text-zinc-900 dark:text-white'
                      : 'text-zinc-600 dark:text-zinc-400'
                  }`}
                >
                  {step.label}
                </span>
                {index < steps.length - 1 && (
                  <div
                    className={`mx-4 h-[2px] w-12 ${
                      step.number < currentStep ? 'bg-lime-500' : 'bg-zinc-200 dark:bg-zinc-800'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Content - Two Column Layout */}
        <div className="grid grid-cols-2 gap-6 p-6" style={{ maxHeight: 'calc(90vh - 200px)' }}>
          {/* Left Column - Wizard Steps */}
          <div className="overflow-y-auto">
            {currentStep === 1 && (
              <StepBasicInfo
                data={wizardData}
                onChange={updateWizardData}
                onNext={handleNext}
                onPrev={handlePrev}
              />
            )}
            {currentStep === 2 && (
              <StepContainer
                data={wizardData}
                onChange={updateWizardData}
                onNext={handleNext}
                onPrev={handlePrev}
              />
            )}
            {currentStep === 3 && (
              <StepResources
                data={wizardData}
                onChange={updateWizardData}
                onNext={handleNext}
                onPrev={handlePrev}
              />
            )}
            {currentStep === 4 && (
              <StepReview
                data={wizardData}
                onChange={updateWizardData}
                onNext={handleNext}
                onPrev={handlePrev}
                onSubmit={handleSubmit}
              />
            )}
          </div>

          {/* Right Column - YAML Preview */}
          <div className="overflow-hidden">
            <YamlPreview data={wizardData} />
          </div>
        </div>
      </div>
    </div>
  );
}
