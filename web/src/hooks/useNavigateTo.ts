import { NavigateOptions, useNavigate } from "react-router-dom";

const useNavigateTo = () => {
  const navigateTo = useNavigate();

  const navigateToWithViewTransition = (to: string, options?: NavigateOptions) => {
    const doc = window.document as unknown as Document & { startViewTransition?: (callback: () => void) => void };
    if (!doc.startViewTransition || doc.visibilityState === "hidden") {
      navigateTo(to, options);
    } else {
      try {
        document.startViewTransition(() => {
          navigateTo(to, options);
        });
      } catch {
        // Fallback if view transition is skipped or fails
        navigateTo(to, options);
      }
    }
  };

  return navigateToWithViewTransition;
};

export default useNavigateTo;
