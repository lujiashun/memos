import { LoaderIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { Link } from "react-router-dom";
import AuthFooter from "@/components/AuthFooter";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authServiceClient } from "@/connect";
import { useInstance } from "@/contexts/InstanceContext";
import useLoading from "@/hooks/useLoading";
import useNavigateTo from "@/hooks/useNavigateTo";
import { handleError } from "@/lib/error";
import { VerificationPurpose } from "@/types/proto/api/v1/auth_service_pb";

const ForgotPassword = () => {
  const navigateTo = useNavigateTo();
  const verifyBtnLoadingState = useLoading(false);
  const resetBtnLoadingState = useLoading(false);
  const [phoneNumber, setPhoneNumber] = useState("");
  const [verificationCode, setVerificationCode] = useState("");
  const [verificationId, setVerificationId] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [isPhoneVerified, setIsPhoneVerified] = useState(false);
  const [verificationMethod, setVerificationMethod] = useState("phone"); // phone or sms
  const { generalSetting: instanceGeneralSetting } = useInstance();

  const handlePhoneInputChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value as string;
    setPhoneNumber(text);
  };

  const handlePasswordInputChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value as string;
    setNewPassword(text);
  };

  const handleVerificationCodeInputChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value as string;
    setVerificationCode(text);
  };

  const handleVerifyPhone = async () => {
    if (phoneNumber === "") {
      return;
    }

    if (verifyBtnLoadingState.isLoading) {
      return;
    }

    try {
      verifyBtnLoadingState.setLoading();
      
      if (verificationMethod === "sms") {
        // 发送短信验证码
        const response = await authServiceClient.sendVerificationCode({
          phoneNumber,
          purpose: VerificationPurpose.FORGOT_PASSWORD,
        });
        if (response.success) {
          toast.success("Verification code sent successfully!");
          // 验证码会通过短信发送给用户，用户需要输入收到的验证码
        }
      } else {
        // 号码认证
        // 这里使用模拟的authToken，实际应该从第三方号码认证服务获取
        const authToken = "mock_auth_token";
        const response = await authServiceClient.verifyPhone({
          phoneNumber,
          purpose: VerificationPurpose.FORGOT_PASSWORD,
          authToken,
        });
        if (response.valid && response.verificationId) {
          setVerificationId(response.verificationId);
          setIsPhoneVerified(true);
          toast.success("Phone verified successfully!");
        }
      }
    } catch (error: unknown) {
      handleError(error, toast.error, {
        fallbackMessage: "Phone verification failed",
      });
    } finally {
      verifyBtnLoadingState.setFinish();
    }
  };

  const handleVerifySMSCode = async () => {
    if (phoneNumber === "" || verificationCode === "") {
      return;
    }

    if (verifyBtnLoadingState.isLoading) {
      return;
    }

    try {
      verifyBtnLoadingState.setLoading();
      const response = await authServiceClient.verifyPhone({
        phoneNumber,
        purpose: VerificationPurpose.FORGOT_PASSWORD,
        authToken: verificationCode, // 使用验证码作为authToken
      });
      if (response.valid && response.verificationId) {
        setVerificationId(response.verificationId);
        setIsPhoneVerified(true);
        toast.success("Verification code verified successfully!");
      }
    } catch (error: unknown) {
      handleError(error, toast.error, {
        fallbackMessage: "Verification code verification failed",
      });
    } finally {
      verifyBtnLoadingState.setFinish();
    }
  };

  const handleResetPassword = async () => {
    if (newPassword === "") {
      return;
    }

    if (resetBtnLoadingState.isLoading) {
      return;
    }

    try {
      resetBtnLoadingState.setLoading();
      await authServiceClient.resetPassword({
        phoneNumber,
        verification: {
          case: "verificationId",
          value: verificationId,
        },
        newPassword,
      });
      toast.success("Password reset successfully!");
      // 重置成功后跳转到登录页面
      navigateTo("/auth");
    } catch (error: unknown) {
      handleError(error, toast.error, {
        fallbackMessage: "Password reset failed",
      });
    } finally {
      resetBtnLoadingState.setFinish();
    }
  };

  return (
    <div className="py-4 sm:py-8 w-80 max-w-full min-h-svh mx-auto flex flex-col justify-start items-center">
      <div className="w-full py-4 grow flex flex-col justify-center items-center">
        <div className="w-full flex flex-row justify-center items-center mb-6">
          <img className="h-14 w-auto rounded-full shadow" src={instanceGeneralSetting.customProfile?.logoUrl || "/logo.webp"} alt="" />
          <p className="ml-2 text-5xl text-foreground opacity-80">{instanceGeneralSetting.customProfile?.title || "Memos"}</p>
        </div>
        <p className="w-full text-2xl mt-2 text-muted-foreground">Reset Your Password</p>
        <div className="w-full mt-2">
          <div className="flex flex-col justify-start items-start w-full gap-4">
            <div className="w-full flex flex-col justify-start items-start">
                  <span className="leading-8 text-muted-foreground">Phone Number</span>
                  <div className="w-full flex gap-2">
                    <Input
                      className="flex-1 bg-background h-10"
                      type="tel"
                      readOnly={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading}
                      placeholder="Phone number"
                      value={phoneNumber}
                      autoComplete="tel"
                      onChange={handlePhoneInputChanged}
                    />
                  </div>
                </div>
                <div className="w-full flex flex-col justify-start items-start">
                  <span className="leading-8 text-muted-foreground">Verification Method</span>
                  <div className="w-full flex gap-2">
                    <Button 
                      type="button" 
                      className={`h-10 px-4 ${verificationMethod === "phone" ? "bg-primary" : "bg-background"}`}
                      disabled={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading}
                      onClick={() => setVerificationMethod("phone")}
                    >
                      Phone Auth
                    </Button>
                    <Button 
                      type="button" 
                      className={`h-10 px-4 ${verificationMethod === "sms" ? "bg-primary" : "bg-background"}`}
                      disabled={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading}
                      onClick={() => setVerificationMethod("sms")}
                    >
                      SMS Code
                    </Button>
                  </div>
                </div>
                {verificationMethod === "sms" && (
                  <div className="w-full flex flex-col justify-start items-start">
                    <span className="leading-8 text-muted-foreground">Verification Code</span>
                    <div className="w-full flex gap-2">
                      <Input
                        className="flex-1 bg-background h-10"
                        type="text"
                        readOnly={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading}
                        placeholder="Verification code"
                        value={verificationCode}
                        onChange={handleVerificationCodeInputChanged}
                      />
                      <Button 
                        type="button" 
                        className="h-10 px-4"
                        disabled={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading || phoneNumber === ""}
                        onClick={handleVerifyPhone}
                      >
                        Send Code
                        {verifyBtnLoadingState.isLoading && <LoaderIcon className="w-4 h-auto ml-2 animate-spin opacity-60" />}
                      </Button>
                    </div>
                    {verificationCode !== "" && (
                      <Button 
                        type="button" 
                        className="h-10 px-4 mt-2"
                        disabled={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading || verificationCode === ""}
                        onClick={handleVerifySMSCode}
                      >
                        Verify Code
                        {verifyBtnLoadingState.isLoading && <LoaderIcon className="w-4 h-auto ml-2 animate-spin opacity-60" />}
                      </Button>
                    )}
                  </div>
                )}
                {verificationMethod === "phone" && (
                  <div className="w-full flex flex-col justify-start items-start">
                    <Button 
                      type="button" 
                      className="h-10 px-4"
                      disabled={verifyBtnLoadingState.isLoading || resetBtnLoadingState.isLoading || phoneNumber === ""}
                      onClick={handleVerifyPhone}
                    >
                      {isPhoneVerified ? "Verified" : "Verify Phone"}
                      {verifyBtnLoadingState.isLoading && <LoaderIcon className="w-4 h-auto ml-2 animate-spin opacity-60" />}
                    </Button>
                  </div>
                )}
            <div className="w-full flex flex-col justify-start items-start">
              <span className="leading-8 text-muted-foreground">New Password</span>
              <Input
                className="w-full bg-background h-10"
                type="password"
                readOnly={resetBtnLoadingState.isLoading || !isPhoneVerified}
                placeholder="New password"
                value={newPassword}
                autoComplete="new-password"
                onChange={handlePasswordInputChanged}
              />
            </div>
          </div>
          <div className="flex flex-row justify-end items-center w-full mt-6">
            <Button 
              type="button" 
              className="w-full h-10" 
              disabled={resetBtnLoadingState.isLoading || !isPhoneVerified || newPassword === ""}
              onClick={handleResetPassword}
            >
              Reset Password
              {resetBtnLoadingState.isLoading && <LoaderIcon className="w-5 h-auto ml-2 animate-spin opacity-60" />}
            </Button>
          </div>
          <p className="w-full mt-4 text-sm">
            <span className="text-muted-foreground">Remember your password?</span>
            <Link to="/auth" className="cursor-pointer ml-2 text-primary hover:underline" viewTransition>
              Sign in
            </Link>
          </p>
        </div>
      </div>
      <AuthFooter />
    </div>
  );
};

export default ForgotPassword;