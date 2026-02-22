import { create } from "@bufbuild/protobuf";
import { format } from "date-fns";
import { ChevronDown, ChevronRight, CopyIcon, LoaderIcon, SparklesIcon } from "lucide-react";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import MemoContent from "@/components/MemoContent";
import MobileHeader from "@/components/MobileHeader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { memoServiceClient } from "@/connect";
import { useAuth } from "@/contexts/AuthContext";
import { useUpdateUserGeneralSetting } from "@/hooks/useUserQueries";
import { GetMemoInsightRequestSchema } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";

const Insight = () => {
  const t = useTranslate();
  const { currentUser, userGeneralSetting, refetchSettings } = useAuth();
  const { mutateAsync: updateGeneralSetting } = useUpdateUserGeneralSetting(currentUser?.name);
  const [startDate, setStartDate] = useState(format(new Date(), "yyyy-MM-dd"));
  const [endDate, setEndDate] = useState(format(new Date(), "yyyy-MM-dd"));
  const [tags, setTags] = useState("");
  const DEFAULT_PROMPT = "Please review the following memos and generate a concise summary:";
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [showPrompt, setShowPrompt] = useState(false);
  const [insight, setInsight] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const savedPrompt = userGeneralSetting?.memoInsightPrompt || localStorage.getItem("insight-custom-prompt") || DEFAULT_PROMPT;
    setPrompt(savedPrompt);
  }, [userGeneralSetting?.memoInsightPrompt]);

  const handleSavePrompt = async () => {
    localStorage.setItem("insight-custom-prompt", prompt);
    if (currentUser?.name) {
      try {
        await updateGeneralSetting({
          generalSetting: { memoInsightPrompt: prompt },
          updateMask: ["memo_insight_prompt"],
        });
        await refetchSettings();
      } catch (error) {
        console.error(error);
        toast.error("Failed to save prompt");
        return;
      }
    }
    toast.success("Prompt saved");
  };

  const handleGenerate = async () => {
    setIsLoading(true);
    try {
      const conditions = [];
      if (startDate) {
        const startTs = Math.floor(new Date(startDate).getTime() / 1000);
        conditions.push(`created_ts >= ${startTs}`);
      }
      if (endDate) {
        // End date is inclusive for the day, so +1 day
        const endTs = Math.floor(new Date(endDate).getTime() / 1000) + 24 * 60 * 60;
        conditions.push(`created_ts < ${endTs}`);
      }
      if (tags.trim()) {
        const tagList = tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean);
        if (tagList.length > 0) {
          const tagListString = tagList.map((t) => `"${t}"`).join(", ");
          conditions.push(`tag in [${tagListString}]`);
        }
      }

      const filter = conditions.join(" && ");

      const response = await memoServiceClient.getMemoInsight(
        create(GetMemoInsightRequestSchema, {
          filter: filter,
          prompt: prompt,
        }),
      );
      setInsight(response.content);
    } catch (error) {
      console.error(error);
      toast.error("Failed to generate insight");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <section className="@container w-full max-w-5xl min-h-full flex flex-col justify-start items-start sm:pt-3 md:pt-6 pb-8">
      <MobileHeader />
      <div className="w-full px-4 sm:px-6">
        <div className="flex flex-col gap-4 max-w-2xl mx-auto">
          <h1 className="text-3xl font-bold">{t("insight.title")}</h1>
          <div className="flex flex-col gap-4 p-4 border rounded-lg shadow-sm bg-white dark:bg-zinc-800">
            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-2">
                <label className="text-sm font-medium">{t("insight.start-date")}</label>
                <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
              </div>
              <div className="flex flex-col gap-2">
                <label className="text-sm font-medium">{t("insight.end-date")}</label>
                <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium">{t("insight.tags")}</label>
              <Input type="text" placeholder="tag1, tag2" value={tags} onChange={(e) => setTags(e.target.value)} />
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-1 cursor-pointer select-none" onClick={() => setShowPrompt(!showPrompt)}>
                {showPrompt ? <ChevronDown className="w-4 h-4 ml-[-4px]" /> : <ChevronRight className="w-4 h-4 ml-[-4px]" />}
                <label className="text-sm font-medium cursor-pointer">{t("insight.customize-prompt")}</label>
              </div>
              {showPrompt && (
                <div className="flex flex-col gap-2">
                  <Textarea
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder={t("insight.prompt")}
                    className="min-h-[100px]"
                  />
                  <div className="flex justify-end">
                    <Button variant="outline" size="sm" onClick={handleSavePrompt}>
                      {t("common.save")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
            <Button onClick={handleGenerate} disabled={isLoading} className="w-full">
              {isLoading ? <LoaderIcon className="w-5 h-5 animate-spin" /> : <SparklesIcon className="w-5 h-5 mr-2" />}
              {t("insight.generate")}
            </Button>
          </div>

          {insight && (
            <div className="p-4 border rounded-lg bg-white dark:bg-zinc-800">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-xl font-semibold">{t("insight.result")}</h2>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(insight);
                    toast.success("Copied to clipboard");
                  }}
                >
                  <CopyIcon className="w-4 h-4" />
                </Button>
              </div>
              <MemoContent content={insight} showCopyOnTranscript />
            </div>
          )}
        </div>
      </div>
    </section>
  );
};

export default Insight;
