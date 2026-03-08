import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useInstance } from "@/contexts/InstanceContext";
import { handleError } from "@/lib/error";
import {
  InstanceSetting_Key,
  InstanceSetting_SmsSetting,
  InstanceSettingSchema,
} from "@/types/proto/api/v1/instance_service_pb";
import { useTranslate } from "@/utils/i18n";
import SettingGroup from "./SettingGroup";
import SettingRow from "./SettingRow";
import SettingSection from "./SettingSection";

const SMSSection = () => {
  const t = useTranslate();
  const { fetchSetting, updateSetting } = useInstance();
  const [smsSetting, setSmsSetting] = useState<InstanceSetting_SmsSetting>({
    verificationMethod: "phone",
    apiKey: "",
    apiSecret: "",
    templateId: "",
    endpoint: "",
    signName: "",
    $typeName: "memos.api.v1.InstanceSetting.SmsSetting",
  });
  const [originalSetting, setOriginalSetting] = useState<InstanceSetting_SmsSetting>(smsSetting);

  useEffect(() => {
    // Fetch the SMS setting when the component mounts
    (async () => {
      try {
        const setting = await fetchSetting(InstanceSetting_Key.SMS);
        if (setting && setting.value.case === "smsSetting") {
          setSmsSetting(setting.value.value);
          setOriginalSetting(setting.value.value);
        }
      } catch (error: unknown) {
        handleError(error, toast.error, {
          context: "Fetch SMS settings",
        });
      }
    })();
  }, [fetchSetting]);

  const updatePartialSetting = (partial: Partial<InstanceSetting_SmsSetting>) => {
    setSmsSetting({
      ...smsSetting,
      ...partial,
    });
  };

  const handleSaveSmsSetting = async () => {
    try {
      await updateSetting(
        create(InstanceSettingSchema, {
          name: `instance/settings/${InstanceSetting_Key[InstanceSetting_Key.SMS]}`,
          value: {
            case: "smsSetting",
            value: smsSetting,
          },
        }),
      );
      await fetchSetting(InstanceSetting_Key.SMS);
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "Update SMS settings",
      });
      return;
    }
    toast.success(t("message.update-succeed"));
  };

  return (
    <SettingSection>
      <SettingGroup title={t("setting.sms-section.title")}>
        <SettingRow label={t("setting.sms-section.verification-method")}>
          <Select
            value={smsSetting.verificationMethod}
            onValueChange={(value) => updatePartialSetting({ verificationMethod: value })}
          >
            <SelectTrigger className="min-w-fit">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="phone">{t("setting.sms-section.phone-auth")}</SelectItem>
              <SelectItem value="sms">{t("setting.sms-section.sms-code")}</SelectItem>
            </SelectContent>
          </Select>
        </SettingRow>

        <SettingRow label={t("setting.sms-section.api-key")}>
          <Input
            value={smsSetting.apiKey}
            onChange={(event) => updatePartialSetting({ apiKey: event.target.value })}
          />
        </SettingRow>

        <SettingRow label={t("setting.sms-section.api-secret")}>
          <Input
            type="password"
            value={smsSetting.apiSecret}
            onChange={(event) => updatePartialSetting({ apiSecret: event.target.value })}
          />
        </SettingRow>

        <SettingRow label={t("setting.sms-section.template-id")}>
          <Input
            value={smsSetting.templateId}
            onChange={(event) => updatePartialSetting({ templateId: event.target.value })}
          />
        </SettingRow>

        <SettingRow label={t("setting.sms-section.sign-name")}>
          <Input
            value={smsSetting.signName}
            onChange={(event) => updatePartialSetting({ signName: event.target.value })}
            placeholder="e.g., Memos"
          />
        </SettingRow>

        <SettingRow label={t("setting.sms-section.endpoint")}>
          <Input
            value={smsSetting.endpoint}
            onChange={(event) => updatePartialSetting({ endpoint: event.target.value })}
            placeholder="e.g., dypnsapi.aliyuncs.com"
          />
        </SettingRow>
      </SettingGroup>

      <div className="w-full flex justify-end">
        <Button disabled={isEqual(smsSetting, originalSetting)} onClick={handleSaveSmsSetting}>
          {t("common.save")}
        </Button>
      </div>
    </SettingSection>
  );
};

export default SMSSection;