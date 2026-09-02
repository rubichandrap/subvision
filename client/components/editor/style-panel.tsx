'use client';

import * as React from 'react';
import { PipetteIcon } from 'lucide-react';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import {
  COLOR_SWATCHES,
  FONT_FAMILIES,
  type SubtitleStyleState,
} from '@/lib/edit-spec';

// The Subtitle Style panel: every knob the renderer honors, previewed live on
// the stage. The font list matches the faces bundled into the vfx service.

export interface StylePanelProps {
  style: SubtitleStyleState;
  animation: string;
  onChange: (style: SubtitleStyleState) => void;
  disabled?: boolean;
}

function SwatchRow({
  value,
  onChange,
  disabled,
  ariaLabel,
}: {
  value: string;
  onChange: (color: string) => void;
  disabled?: boolean;
  ariaLabel: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      {COLOR_SWATCHES.map((color) => (
        <button
          key={color}
          type="button"
          disabled={disabled}
          aria-label={`${ariaLabel}: ${color}`}
          onClick={() => onChange(color)}
          className={`h-6 w-6 rounded-md border transition-transform disabled:opacity-50 ${
            value.toLowerCase() === color.toLowerCase()
              ? 'scale-110 border-foreground'
              : 'border-border/70 hover:scale-105'
          }`}
          style={{ backgroundColor: color }}
        />
      ))}
      <label
        className="relative flex h-6 w-6 cursor-pointer items-center justify-center rounded-md border border-border/70 text-muted-foreground hover:scale-105 disabled:opacity-50"
        title="Custom color"
      >
        <PipetteIcon className="h-3.5 w-3.5" />
        <input
          type="color"
          value={value}
          disabled={disabled}
          aria-label={`Custom ${ariaLabel}`}
          onChange={(event) => onChange(event.target.value.toUpperCase())}
          className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
        />
      </label>
    </div>
  );
}

function FieldRow({
  label,
  value,
  children,
}: {
  label: string;
  value?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="flex items-baseline justify-between">
        <Label className="text-sm">{label}</Label>
        {value && (
          <span className="text-xs tabular-nums text-muted-foreground">{value}</span>
        )}
      </div>
      <div className="mt-2">{children}</div>
    </div>
  );
}

export function StylePanel({ style, animation, onChange, disabled }: StylePanelProps) {
  const patch = (changes: Partial<SubtitleStyleState>) =>
    onChange({ ...style, ...changes });
  const highlightRelevant = animation === 'karaoke' || animation === 'pop' || animation === 'random';

  return (
    <div className="space-y-5">
      <FieldRow label="Font">
        <Select
          value={style.fontFamily}
          onValueChange={(fontFamily) =>
            patch({ fontFamily: fontFamily as SubtitleStyleState['fontFamily'] })
          }
          disabled={disabled}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FONT_FAMILIES.map((font) => (
              <SelectItem key={font} value={font}>
                {font}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FieldRow>

      <FieldRow label="Size" value={`${Math.round(style.fontSizeScale * 100)}%`}>
        <Slider
          value={[style.fontSizeScale]}
          min={0.5}
          max={2}
          step={0.05}
          disabled={disabled}
          aria-label="Caption size"
          onValueChange={([fontSizeScale]) => patch({ fontSizeScale })}
        />
      </FieldRow>

      <FieldRow label="Color">
        <SwatchRow
          value={style.color}
          onChange={(color) => patch({ color })}
          disabled={disabled}
          ariaLabel="Caption color"
        />
      </FieldRow>

      <FieldRow
        label="Outline"
        value={style.outlineWidth === 0 ? 'off' : `${Math.round(style.outlineWidth)} px`}
      >
        <Slider
          value={[style.outlineWidth]}
          min={0}
          max={32}
          step={1}
          disabled={disabled}
          aria-label="Outline width"
          onValueChange={([outlineWidth]) => patch({ outlineWidth })}
        />
        <div className="mt-2.5">
          <SwatchRow
            value={style.outlineColor}
            onChange={(outlineColor) => patch({ outlineColor })}
            disabled={disabled || style.outlineWidth === 0}
            ariaLabel="Outline color"
          />
        </div>
      </FieldRow>

      <FieldRow
        label="Position"
        value={
          style.bottomMargin <= 0.1
            ? 'Low'
            : style.bottomMargin <= 0.35
              ? 'Lower middle'
              : 'Middle'
        }
      >
        <Slider
          value={[style.bottomMargin]}
          min={0}
          max={0.8}
          step={0.01}
          disabled={disabled}
          aria-label="Caption vertical position"
          onValueChange={([bottomMargin]) => patch({ bottomMargin })}
        />
      </FieldRow>

      <FieldRow label="Background">
        <div className="flex items-center gap-3">
          <Select
            value={style.background}
            onValueChange={(background) =>
              patch({ background: background as SubtitleStyleState['background'] })
            }
            disabled={disabled}
          >
            <SelectTrigger className="w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">None</SelectItem>
              <SelectItem value="box">Box</SelectItem>
            </SelectContent>
          </Select>
          {style.background === 'box' && (
            <Slider
              value={[style.backgroundOpacity]}
              min={0.1}
              max={1}
              step={0.05}
              disabled={disabled}
              aria-label="Background opacity"
              onValueChange={([backgroundOpacity]) => patch({ backgroundOpacity })}
              className="flex-1"
            />
          )}
        </div>
      </FieldRow>

      <div className="flex items-center justify-between">
        <Label className="text-sm" htmlFor="uppercase-toggle">
          ALL CAPS
        </Label>
        <Switch
          id="uppercase-toggle"
          checked={style.uppercase}
          onCheckedChange={(uppercase) => patch({ uppercase })}
          disabled={disabled}
        />
      </div>

      {highlightRelevant && (
        <FieldRow label="Highlight (karaoke & pop)">
          <SwatchRow
            value={style.highlightColor}
            onChange={(highlightColor) => patch({ highlightColor })}
            disabled={disabled}
            ariaLabel="Highlight color"
          />
        </FieldRow>
      )}
    </div>
  );
}
