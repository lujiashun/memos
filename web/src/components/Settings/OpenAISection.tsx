import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useInstance } from "@/contexts/InstanceContext";
import { handleError } from "@/lib/error";
import {
  InstanceSetting_Key,
  InstanceSetting_OpenAISetting,
  InstanceSetting_OpenAISettingSchema,
  InstanceSettingSchema,
} from "@/types/proto/api/v1/instance_service_pb";
import { useTranslate } from "@/utils/i18n";
import SettingGroup from "./SettingGroup";
import SettingRow from "./SettingRow";
import SettingSection from "./SettingSection";

const OpenAISection = () => {
  const t = useTranslate();
  const { openaiSetting: originalSetting, updateSetting, fetchSetting } = useInstance();
  const [openaiSetting, setOpenaiSetting] = useState<InstanceSetting_OpenAISetting>(originalSetting);

  useEffect(() => {
    fetchSetting(InstanceSetting_Key.OPENAI).catch((error) => {
      console.error("Failed to fetch OpenAI setting:", error);
    });
  }, []);

  useEffect(() => {
    setOpenaiSetting(originalSetting);
  }, [originalSetting]);

  const updatePartialSetting = (partial: Partial<InstanceSetting_OpenAISetting>) => {
    const newInstanceOpenAISetting = create(InstanceSetting_OpenAISettingSchema, {
      ...openaiSetting,
      ...partial,
    });
    setOpenaiSetting(newInstanceOpenAISetting);
  };

  const handleUpdateSetting = async () => {
    if (isEqual(originalSetting, openaiSetting)) {
      return;
    }
    try {
      await updateSetting(
        create(InstanceSettingSchema, {
          name: `instance/settings/${InstanceSetting_Key[InstanceSetting_Key.OPENAI]}`,
          value: {
            case: "openaiSetting",
            value: openaiSetting,
          },
        }),
      );
      await fetchSetting(InstanceSetting_Key.OPENAI);
      toast.success(t("common.saved"));
    } catch (error) {
      handleError(error);
    }
  };

  return (
    <SettingSection title={t("setting.openai")}>
      <SettingGroup>
        <SettingRow
          title="API Key"
          description="The API key for OpenAI service"
          className=""
        >
          <Input
            className="w-full"
            type="password"
            value={openaiSetting.apiKey}
            placeholder="sk-..."
            onChange={(e) => updatePartialSetting({ apiKey: e.target.value })}
          />
        </SettingRow>
        <SettingRow
          title="Base URL"
          description="The base URL for OpenAI service (optional)"
        >
          <Input
            className="w-full"
            value={openaiSetting.baseUrl}
            placeholder="https://api.openai.com/v1"
            onChange={(e) => updatePartialSetting({ baseUrl: e.target.value })}
          />
        </SettingRow>
        <SettingRow
          title="Model"
          description="The model to use (e.g. gpt-4, gpt-3.5-turbo)"
        >
          <Input
            className="w-full"
            value={openaiSetting.model}
            placeholder="gpt-3.5-turbo"
            onChange={(e) => updatePartialSetting({ model: e.target.value })}
          />
        </SettingRow>
      </SettingGroup>
      <div className="mt-4 w-full flex justify-end">
        <Button onClick={handleUpdateSetting} disabled={isEqual(originalSetting, openaiSetting)}>
          {t("common.save")}
        </Button>
      </div>
    </SettingSection>
  );
};

export default OpenAISection;
