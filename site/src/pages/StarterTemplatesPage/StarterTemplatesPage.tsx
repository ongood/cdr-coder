import type { FC } from "react";
import { Helmet } from "react-helmet-async";
import { useQuery } from "react-query";
import { templateExamples } from "api/queries/templates";
import type { TemplateExample } from "api/typesGenerated";
import { useDashboard } from "modules/dashboard/useDashboard";
import { pageTitle } from "utils/page";
import { getTemplatesByTag } from "utils/templateAggregators";
import { StarterTemplatesPageView } from "./StarterTemplatesPageView";

const StarterTemplatesPage: FC = () => {
  const { organizationId } = useDashboard();
  const templateExamplesQuery = useQuery(templateExamples(organizationId));
  const starterTemplatesByTag = templateExamplesQuery.data
    ? // Currently, the scratch template should not be displayed on the starter templates page.
      getTemplatesByTag(removeScratchExample(templateExamplesQuery.data))
    : undefined;

  return (
    <>
      <Helmet>
        <title>{pageTitle("Starter Templates")}</title>
      </Helmet>

      <StarterTemplatesPageView
        error={templateExamplesQuery.error}
        starterTemplatesByTag={starterTemplatesByTag}
      />
    </>
  );
};

const removeScratchExample = (data: TemplateExample[]) => {
  return data.filter((example) => example.id !== "scratch");
};

export default StarterTemplatesPage;
