import { create } from "@bufbuild/protobuf";
import { format } from "date-fns";
import { CopyIcon, LoaderIcon, SparklesIcon } from "lucide-react";
import { useState } from "react";
import toast from "react-hot-toast";
import MemoContent from "@/components/MemoContent";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { memoServiceClient } from "@/connect";
import { GetDailyReviewRequestSchema } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";

interface Props {
  trigger?: React.ReactNode;
}

const DailyReviewDialog = ({ trigger }: Props) => {
  const t = useTranslate();
  const [content, setContent] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);

  const handleFetchReview = async () => {
    setIsLoading(true);
    try {
      const response = await memoServiceClient.getDailyReview(
        create(GetDailyReviewRequestSchema, {
          date: format(new Date(), "yyyy-MM-dd"),
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        }),
      );
      setContent(response.content);
    } catch (error) {
      console.error(error);
      toast.error(t("daily-review.fetch-failed"));
    } finally {
      setIsLoading(false);
    }
  };

  const copyContent = () => {
    navigator.clipboard.writeText(content);
    toast.success(t("common.copied"));
  };

  return (
    <Dialog>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="ghost" size="icon" title={t("daily-review.title")}>
            <SparklesIcon className="w-5 h-5" />
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>{t("daily-review.title")}</DialogTitle>
          <DialogDescription>{format(new Date(), "yyyy-MM-dd")}</DialogDescription>
        </DialogHeader>
        <div className="flex-1 overflow-auto min-h-[200px] p-2">
          {!content && !isLoading && (
            <div className="flex flex-col items-center justify-center h-full gap-4 py-10">
              <p className="text-muted-foreground">{t("daily-review.empty")}</p>
              <Button onClick={handleFetchReview}>
                <SparklesIcon className="w-4 h-4 mr-2" />
                {t("daily-review.generate")}
              </Button>
            </div>
          )}
          {isLoading && (
            <div className="flex flex-col items-center justify-center h-full gap-2 py-10">
              <LoaderIcon className="w-8 h-8 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">{t("daily-review.generating")}</p>
            </div>
          )}
          {content && (
             <div className="w-full">
                 <MemoContent content={content} />
             </div>
          )}
        </div>
        {content && (
            <div className="flex justify-end gap-2 pt-4 border-t">
                <Button variant="outline" onClick={copyContent}>
                    <CopyIcon className="w-4 h-4 mr-2" />
              {t("common.copy")}
                </Button>
            </div>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default DailyReviewDialog;
