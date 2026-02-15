package cmd

import (
	"context"

	"github.com/google/uuid"
	docapp "github.com/sam8helloworld/tms-poc/internal/document/application/document"
	docdomain "github.com/sam8helloworld/tms-poc/internal/document/domain/document"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/spf13/cobra"
)

var documentCmd = &cobra.Command{
	Use:   "document",
	Short: "Manage documents",
}

var documentUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a new document",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipmentID, _ := cmd.Flags().GetString("shipment-id")
		docType, _ := cmd.Flags().GetString("doc-type")
		origin, _ := cmd.Flags().GetString("origin")
		fileName, _ := cmd.Flags().GetString("file-name")
		storageURI, _ := cmd.Flags().GetString("storage-uri")
		uploadedBy, _ := cmd.Flags().GetString("uploaded-by")

		input := docapp.UploadDocumentInput{
			ShipmentID: uuid.MustParse(shipmentID),
			DocType:    shared.DocType(docType),
			Origin:     docdomain.DocumentOrigin(origin),
			FileName:   fileName,
			StorageURI: storageURI,
			UploadedBy: uuid.MustParse(uploadedBy),
		}

		output, err := deps.UploadDocumentUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var documentExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract structured content from a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		documentID, _ := cmd.Flags().GetString("document-id")

		input := docapp.ExtractDocumentContentInput{
			DocumentID: uuid.MustParse(documentID),
		}

		output, err := deps.ExtractContentUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var documentConfirmCmd = &cobra.Command{
	Use:   "confirm",
	Short: "Confirm a document",
	RunE: func(cmd *cobra.Command, args []string) error {
		documentID, _ := cmd.Flags().GetString("document-id")

		input := docapp.ConfirmDocumentInput{
			DocumentID: uuid.MustParse(documentID),
		}

		output, err := deps.ConfirmDocumentUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var documentGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a document by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.DocumentQuery.GetDocument(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var documentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documents by shipment ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipmentID, _ := cmd.Flags().GetString("shipment-id")
		result, err := deps.DocumentQuery.ListDocumentsByShipment(context.Background(), uuid.MustParse(shipmentID))
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	documentUploadCmd.Flags().String("shipment-id", "", "Shipment UUID")
	documentUploadCmd.Flags().String("doc-type", "", "Document type (INVOICE, BL, etc.)")
	documentUploadCmd.Flags().String("origin", "SHIPPER", "Document origin (SHIPPER or PROVIDER)")
	documentUploadCmd.Flags().String("file-name", "", "File name")
	documentUploadCmd.Flags().String("storage-uri", "", "Storage URI")
	documentUploadCmd.Flags().String("uploaded-by", "", "Uploader UUID")
	documentUploadCmd.MarkFlagRequired("shipment-id")
	documentUploadCmd.MarkFlagRequired("doc-type")
	documentUploadCmd.MarkFlagRequired("file-name")
	documentUploadCmd.MarkFlagRequired("storage-uri")
	documentUploadCmd.MarkFlagRequired("uploaded-by")

	documentExtractCmd.Flags().String("document-id", "", "Document UUID")
	documentExtractCmd.MarkFlagRequired("document-id")

	documentConfirmCmd.Flags().String("document-id", "", "Document UUID")
	documentConfirmCmd.MarkFlagRequired("document-id")

	documentListCmd.Flags().String("shipment-id", "", "Shipment UUID")
	documentListCmd.MarkFlagRequired("shipment-id")

	documentCmd.AddCommand(documentUploadCmd)
	documentCmd.AddCommand(documentExtractCmd)
	documentCmd.AddCommand(documentConfirmCmd)
	documentCmd.AddCommand(documentGetCmd)
	documentCmd.AddCommand(documentListCmd)
}
