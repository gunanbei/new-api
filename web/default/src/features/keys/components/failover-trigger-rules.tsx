import { Info } from "lucide-react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import type { ApiKeyFormValues } from "../lib";

const retryFields = [
  {
    name: "retry_on_transport_error",
    label: "Transport errors",
    description:
      "Retries when the request cannot complete because the upstream connection fails.",
    example:
      "Example: DNS lookup failure, connection refused, TLS handshake failure, or connection reset.",
  },
  {
    name: "retry_on_empty_response",
    label: "Empty responses",
    description:
      "Retries when the upstream request succeeds but returns no usable response content.",
    example:
      "Example: HTTP 200 with an empty body, or a stream that ends before any valid content arrives.",
  },
  {
    name: "retry_on_invalid_response",
    label: "Invalid responses",
    description:
      "Retries when the upstream response cannot be parsed as the expected API protocol.",
    example:
      "Example: malformed JSON, a missing required response structure, or invalid SSE data.",
  },
  {
    name: "retry_on_stream_error",
    label: "Interrupted streams",
    description:
      "Retries when a streaming response starts but ends abnormally before completion.",
    example:
      "Example: the SSE connection drops, scanning fails, or the upstream closes without a normal completion event.",
  },
] as const;

const timeoutGroups = [
  {
    title: "Streaming requests",
    headerField: "stream_response_header_timeout_seconds",
    contentField: "stream_first_content_timeout_seconds",
  },
  {
    title: "Non-streaming requests",
    headerField: "non_stream_response_header_timeout_seconds",
    contentField: "non_stream_first_content_timeout_seconds",
  },
] as const;

function RetryRuleHelp({
  label,
  description,
  example,
}: {
  label: string;
  description: string;
  example: string;
}) {
  const { t } = useTranslation();
  const trigger = (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:ring-ring size-6 shrink-0 rounded-full p-0 focus-visible:ring-2"
      aria-label={`${t(label)}: ${t("Learn more")}`}
    >
      <Info className="size-3.5" aria-hidden="true" />
    </Button>
  );

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger render={<PopoverTrigger render={trigger} />} />
        <TooltipContent
          side="top"
          align="start"
          className="max-w-xs flex-col items-start gap-2 text-left"
        >
          <p className="font-medium">{t(label)}</p>
          <p className="leading-relaxed opacity-90">{t(description)}</p>
          <p className="leading-relaxed opacity-75">{t(example)}</p>
        </TooltipContent>
      </Tooltip>
      <PopoverContent
        align="start"
        side="top"
        sideOffset={8}
        collisionPadding={12}
        className="w-[min(20rem,calc(100vw-2rem))] gap-2.5 p-3"
      >
        <div className="text-sm font-medium">{t(label)}</div>
        <p className="text-muted-foreground text-xs leading-relaxed">
          {t(description)}
        </p>
        <div className="bg-muted text-foreground rounded-md px-2.5 py-2 text-xs leading-relaxed">
          {t(example)}
        </div>
      </PopoverContent>
    </Popover>
  );
}

type FailoverTriggerRulesProps = {
  relayTimeout: number;
  automaticRetryStatusCodes: string;
};

export function FailoverTriggerRules({
  relayTimeout,
  automaticRetryStatusCodes,
}: FailoverTriggerRulesProps) {
  const { t } = useTranslation();
  const form = useFormContext<ApiKeyFormValues>();

  return (
    <div className="space-y-4 border-t pt-4">
      <div>
        <h4 className="text-sm font-medium">{t("Failover triggers")}</h4>
        <p className="text-muted-foreground text-xs">
          {t("Choose which upstream failures can start another attempt.")}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {timeoutGroups.map((group) => (
          <div key={group.title} className="space-y-3 border-l-2 pl-3">
            <h5 className="text-sm font-medium">{t(group.title)}</h5>
            <FormField
              control={form.control}
              name={group.headerField}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t("Response header timeout (seconds)")}
                  </FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type="number"
                      min={0}
                      step={1}
                      value={field.value ?? ""}
                      onChange={(event) =>
                        field.onChange(
                          event.target.value === ""
                            ? ""
                            : Number(event.target.value),
                        )
                      }
                      onBlur={(event) => {
                        if (event.target.value === "") {
                          field.onChange(0);
                        }
                        field.onBlur();
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={group.contentField}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t("First valid content timeout (seconds)")}
                  </FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type="number"
                      min={0}
                      step={1}
                      value={field.value ?? ""}
                      onChange={(event) =>
                        field.onChange(
                          event.target.value === ""
                            ? ""
                            : Number(event.target.value),
                        )
                      }
                      onBlur={(event) => {
                        if (event.target.value === "") {
                          field.onChange(0);
                        }
                        field.onBlur();
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        ))}
      </div>
      {relayTimeout > 0 && (
        <p className="text-muted-foreground text-xs">
          {t("Global timeout of {{timeout}} seconds still applies.", {
            timeout: relayTimeout,
          })}
        </p>
      )}

      <FormField
        control={form.control}
        name="failover_http_status_codes"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("Retry HTTP status codes")}</FormLabel>
            <FormControl>
              <Input {...field} placeholder="401,429,500-599" />
            </FormControl>
            <FormDescription>
              {t(
                "Separate status codes with commas and use a hyphen for inclusive ranges, e.g. 401,429,500-599.",
              )}
            </FormDescription>
            <FormDescription>
              {t(
                "System retry status code rules [{{rules}}] always apply. Add extra status codes, but they cannot override system rules.",
                { rules: automaticRetryStatusCodes || '-' },
              )}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className="grid gap-2 sm:grid-cols-2">
        {retryFields.map(({ name, label, description, example }) => (
          <FormField
            key={name}
            control={form.control}
            name={name}
            render={({ field }) => (
              <FormItem className="flex min-h-10 items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-1">
                  <FormLabel className="text-sm font-normal">
                    {t(label)}
                  </FormLabel>
                  <RetryRuleHelp
                    label={label}
                    description={description}
                    example={example}
                  />
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />
        ))}
      </div>
    </div>
  );
}
