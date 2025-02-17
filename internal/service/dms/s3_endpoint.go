package dms

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func ResourceS3Endpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceS3EndpointCreate,
		Read:   resourceS3EndpointRead,
		Update: resourceS3EndpointUpdate,
		Delete: resourceS3EndpointDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"certificate_arn": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"endpoint_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validEndpointID,
			},
			"endpoint_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(dms.ReplicationEndpointTypeValue_Values(), false),
			},
			"engine_display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"external_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"ssl_mode": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.DmsSslModeValue_Values(), false),
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),

			/////// S3-Specific Settings
			"add_column_name": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"add_trailing_padding_character": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"bucket_folder": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"bucket_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"canned_acl_for_objects": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.CannedAclForObjectsValue_Values(), true),
				StateFunc: func(v interface{}) string {
					return strings.ToLower(v.(string))
				},
			},
			"cdc_inserts_and_updates": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cdc_inserts_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cdc_max_batch_interval": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"cdc_min_file_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"cdc_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"compression_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.CompressionTypeValue_Values(), true),
				Default:      strings.ToUpper(dms.CompressionTypeValueNone),
				StateFunc: func(v interface{}) string {
					return strings.ToUpper(v.(string))
				},
			},
			"csv_delimiter": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ",",
			},
			"csv_no_sup_value": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"csv_null_value": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"csv_row_delimiter": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "\\n",
			},
			"data_format": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.DataFormatValue_Values(), false),
			},
			"data_page_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"date_partition_delimiter": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.DatePartitionDelimiterValue_Values(), true),
				StateFunc: func(v interface{}) string {
					return strings.ToUpper(v.(string))
				},
			},
			"date_partition_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"date_partition_sequence": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.DatePartitionSequenceValue_Values(), true),
				StateFunc: func(v interface{}) string {
					return strings.ToLower(v.(string))
				},
			},
			"date_partition_timezone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dict_page_size_limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"enable_statistics": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"encoding_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.EncodingTypeValue_Values(), false),
			},
			"encryption_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(encryptionMode_Values(), false),
			},
			"expected_bucket_owner": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidAccountID,
			},
			"external_table_definition": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsJSON,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"ignore_header_rows": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntInSlice([]int{0, 1}),
			},
			"include_op_for_full_load": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"max_file_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 1048576),
			},
			"parquet_timestamp_in_millisecond": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"parquet_version": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(dms.ParquetVersionValue_Values(), false),
			},
			"preserve_transactions": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"rfc_4180": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"row_group_length": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"server_side_encryption_kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"service_access_role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.ValidARN,
			},
			"timestamp_column_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"use_csv_no_sup_value": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"use_task_start_time_for_full_load_timestamp": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

const (
	ResNameS3Endpoint = "S3 Endpoint"
)

func resourceS3EndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	input := &dms.CreateEndpointInput{
		EndpointIdentifier: aws.String(d.Get("endpoint_id").(string)),
		EndpointType:       aws.String(d.Get("endpoint_type").(string)),
		EngineName:         aws.String("s3"),
		Tags:               Tags(tags.IgnoreAWS()),
	}

	if v, ok := d.GetOk("certificate_arn"); ok {
		input.CertificateArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("kms_key_arn"); ok {
		input.KmsKeyId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ssl_mode"); ok {
		input.SslMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_access_role_arn"); ok {
		input.ServiceAccessRoleArn = aws.String(v.(string))
	}

	input.S3Settings = s3Settings(d, d.Get("endpoint_type").(string) == dms.ReplicationEndpointTypeValueTarget)

	log.Println("[DEBUG] DMS create endpoint:", input)

	var out *dms.CreateEndpointOutput
	err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		var err error
		out, err = conn.CreateEndpoint(input)

		if tfawserr.ErrCodeEquals(err, "AccessDeniedFault") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		out, err = conn.CreateEndpoint(input)
	}

	if err != nil || out == nil || out.Endpoint == nil {
		return create.Error(names.DMS, create.ErrActionCreating, ResNameS3Endpoint, d.Get("endpoint_id").(string), err)
	}

	d.SetId(d.Get("endpoint_id").(string))
	d.Set("endpoint_arn", out.Endpoint.EndpointArn)

	// AWS bug? ssekki is ignored on create but sets on update
	if _, ok := d.GetOk("server_side_encryption_kms_key_id"); ok {
		return resourceS3EndpointUpdate(d, meta)
	}

	return resourceS3EndpointRead(d, meta)
}

func resourceS3EndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	endpoint, err := FindEndpointByID(conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] DMS Endpoint (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return create.Error(names.DMS, create.ErrActionReading, ResNameS3Endpoint, d.Id(), err)
	}

	if endpoint.S3Settings == nil {
		return create.Error(names.DMS, create.ErrActionReading, ResNameS3Endpoint, d.Id(), errors.New("no settings returned"))
	}

	d.Set("endpoint_arn", endpoint.EndpointArn)

	d.Set("certificate_arn", endpoint.CertificateArn)
	d.Set("endpoint_id", endpoint.EndpointIdentifier)
	d.Set("endpoint_type", strings.ToLower(*endpoint.EndpointType)) // For some reason the AWS API only accepts lowercase type but returns it as uppercase
	d.Set("engine_display_name", endpoint.EngineDisplayName)
	d.Set("external_id", endpoint.ExternalId)
	// d.Set("external_table_definition", endpoint.ExternalTableDefinition) // set from s3 settings
	d.Set("kms_key_arn", endpoint.KmsKeyId)
	// d.Set("service_access_role_arn", endpoint.ServiceAccessRoleArn) // set from s3 settings
	d.Set("ssl_mode", endpoint.SslMode)
	d.Set("status", endpoint.Status)

	s3settings := endpoint.S3Settings
	d.Set("add_column_name", s3settings.AddColumnName)
	d.Set("bucket_folder", s3settings.BucketFolder)
	d.Set("bucket_name", s3settings.BucketName)
	d.Set("canned_acl_for_objects", s3settings.CannedAclForObjects)
	d.Set("cdc_inserts_and_updates", s3settings.CdcInsertsAndUpdates)
	d.Set("cdc_inserts_only", s3settings.CdcInsertsOnly)
	d.Set("cdc_max_batch_interval", s3settings.CdcMaxBatchInterval)
	d.Set("cdc_min_file_size", s3settings.CdcMinFileSize)
	d.Set("cdc_path", s3settings.CdcPath)
	d.Set("csv_delimiter", s3settings.CsvDelimiter)
	d.Set("csv_null_value", s3settings.CsvNullValue)
	d.Set("csv_row_delimiter", s3settings.CsvRowDelimiter)
	d.Set("data_page_size", s3settings.DataPageSize)
	d.Set("dict_page_size_limit", s3settings.DictPageSizeLimit)
	d.Set("enable_statistics", s3settings.EnableStatistics)
	d.Set("encoding_type", s3settings.EncodingType)
	d.Set("expected_bucket_owner", s3settings.ExpectedBucketOwner)
	d.Set("ignore_header_rows", s3settings.IgnoreHeaderRows)
	d.Set("include_op_for_full_load", s3settings.IncludeOpForFullLoad)
	d.Set("max_file_size", s3settings.MaxFileSize)
	d.Set("rfc_4180", s3settings.Rfc4180)
	d.Set("row_group_length", s3settings.RowGroupLength)
	d.Set("service_access_role_arn", s3settings.ServiceAccessRoleArn)
	d.Set("timestamp_column_name", s3settings.TimestampColumnName)
	d.Set("use_task_start_time_for_full_load_timestamp", s3settings.UseTaskStartTimeForFullLoadTimestamp)

	if d.Get("endpoint_type").(string) == dms.ReplicationEndpointTypeValueTarget {
		d.Set("add_trailing_padding_character", s3settings.AddTrailingPaddingCharacter)
		d.Set("compression_type", s3settings.CompressionType)
		d.Set("csv_no_sup_value", s3settings.CsvNoSupValue)
		d.Set("data_format", s3settings.DataFormat)
		d.Set("date_partition_delimiter", strings.ToUpper(aws.StringValue(s3settings.DatePartitionDelimiter)))
		d.Set("date_partition_enabled", s3settings.DatePartitionEnabled)
		d.Set("date_partition_sequence", s3settings.DatePartitionSequence)
		d.Set("date_partition_timezone", s3settings.DatePartitionTimezone)
		d.Set("encryption_mode", s3settings.EncryptionMode)
		d.Set("parquet_timestamp_in_millisecond", s3settings.ParquetTimestampInMillisecond)
		d.Set("parquet_version", s3settings.ParquetVersion)
		d.Set("preserve_transactions", s3settings.PreserveTransactions)
		d.Set("server_side_encryption_kms_key_id", s3settings.ServerSideEncryptionKmsKeyId)
		d.Set("use_csv_no_sup_value", s3settings.UseCsvNoSupValue)
	}

	p, err := structure.NormalizeJsonString(aws.StringValue(s3settings.ExternalTableDefinition))
	if err != nil {
		return create.Error(names.DMS, create.ErrActionSetting, ResNameS3Endpoint, d.Id(), err)
	}

	d.Set("external_table_definition", p)

	tags, err := ListTags(conn, d.Get("endpoint_arn").(string))
	if err != nil {
		return create.Error(names.DMS, create.ErrActionReading, ResNameS3Endpoint, d.Id(), err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return create.Error(names.DMS, create.ErrActionReading, ResNameS3Endpoint, d.Id(), err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return create.Error(names.DMS, create.ErrActionReading, ResNameS3Endpoint, d.Id(), err)
	}

	return nil
}

func resourceS3EndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn

	if d.HasChangesExcept("tags", "tags_all") {
		input := &dms.ModifyEndpointInput{
			EndpointArn: aws.String(d.Get("endpoint_arn").(string)),
		}

		if d.HasChange("certificate_arn") {
			input.CertificateArn = aws.String(d.Get("certificate_arn").(string))
		}

		if d.HasChange("endpoint_type") {
			input.EndpointType = aws.String(d.Get("endpoint_type").(string))
		}

		input.EngineName = aws.String(engineNameS3)

		if d.HasChange("ssl_mode") {
			input.SslMode = aws.String(d.Get("ssl_mode").(string))
		}

		if d.HasChangesExcept(
			"certificate_arn",
			"endpoint_type",
			"ssl_mode",
		) {
			input.S3Settings = s3Settings(d, d.Get("endpoint_type").(string) == dms.ReplicationEndpointTypeValueTarget)
			input.ServiceAccessRoleArn = aws.String(d.Get("service_access_role_arn").(string))
		}

		log.Println("[DEBUG] DMS update endpoint:", input)

		err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
			_, err := conn.ModifyEndpoint(input)

			if tfawserr.ErrCodeEquals(err, "AccessDeniedFault") {
				return resource.RetryableError(err)
			}

			if err != nil {
				return resource.NonRetryableError(err)
			}

			return nil
		})

		if tfresource.TimedOut(err) {
			_, err = conn.ModifyEndpoint(input)
		}

		if err != nil {
			return create.Error(names.DMS, create.ErrActionUpdating, ResNameS3Endpoint, d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		arn := d.Get("endpoint_arn").(string)
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, arn, o, n); err != nil {
			return create.Error(names.DMS, create.ErrActionUpdating, ResNameS3Endpoint, d.Id(), err)
		}
	}

	return resourceS3EndpointRead(d, meta)
}

func resourceS3EndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn

	log.Printf("[DEBUG] Deleting DMS Endpoint: (%s)", d.Id())
	_, err := conn.DeleteEndpoint(&dms.DeleteEndpointInput{
		EndpointArn: aws.String(d.Get("endpoint_arn").(string)),
	})

	if tfawserr.ErrCodeEquals(err, dms.ErrCodeResourceNotFoundFault) {
		return nil
	}

	if err != nil {
		return create.Error(names.DMS, create.ErrActionDeleting, ResNameS3Endpoint, d.Id(), err)
	}

	if err = waitEndpointDeleted(conn, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return create.Error(names.DMS, create.ErrActionWaitingForDeletion, ResNameS3Endpoint, d.Id(), err)
	}

	return err
}

func s3Settings(d *schema.ResourceData, target bool) *dms.S3Settings {
	s3s := &dms.S3Settings{}

	if v, ok := d.Get("add_column_name").(bool); ok { // likely only useful for target
		s3s.AddColumnName = aws.Bool(v)
	}

	if v, ok := d.Get("add_trailing_padding_character").(bool); ok && target { // target
		s3s.AddTrailingPaddingCharacter = aws.Bool(v)
	}

	if v, ok := d.GetOk("bucket_folder"); ok {
		s3s.BucketFolder = aws.String(v.(string))
	}

	if v, ok := d.GetOk("bucket_name"); ok {
		s3s.BucketName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("canned_acl_for_objects"); ok { // likely only useful for target
		s3s.CannedAclForObjects = aws.String(v.(string))
	}

	if v, ok := d.Get("cdc_inserts_and_updates").(bool); ok { // likely only useful for target
		s3s.CdcInsertsAndUpdates = aws.Bool(v)
	}

	if v, ok := d.Get("cdc_inserts_only").(bool); ok { // likely only useful for target
		s3s.CdcInsertsOnly = aws.Bool(v)
	}

	if v, ok := d.GetOk("cdc_max_batch_interval"); ok { // likely only useful for target
		s3s.CdcMaxBatchInterval = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("cdc_min_file_size"); ok { // likely only useful for target
		s3s.CdcMinFileSize = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("cdc_path"); ok {
		s3s.CdcPath = aws.String(v.(string))
	}

	if v, ok := d.GetOk("compression_type"); ok && target { // likely only useful for target
		s3s.CompressionType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("csv_delimiter"); ok {
		s3s.CsvDelimiter = aws.String(v.(string))
	}

	if v, ok := d.GetOk("csv_no_sup_value"); ok && target { // target
		s3s.CsvNoSupValue = aws.String(v.(string))
	}

	if v, ok := d.GetOk("csv_null_value"); ok { // likely only useful for target
		s3s.CsvNullValue = aws.String(v.(string))
	}

	if v, ok := d.GetOk("csv_row_delimiter"); ok {
		s3s.CsvRowDelimiter = aws.String(v.(string))
	}

	if v, ok := d.GetOk("data_format"); ok && target { // target
		s3s.DataFormat = aws.String(v.(string))
	}

	if v, ok := d.GetOk("data_page_size"); ok { // likely only useful for target
		s3s.DataPageSize = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("date_partition_delimiter"); ok && target { // target
		s3s.DatePartitionDelimiter = aws.String(v.(string))
	}

	if v, ok := d.Get("date_partition_enabled").(bool); ok && target { // likely only useful for target
		s3s.DatePartitionEnabled = aws.Bool(v)
	}

	if v, ok := d.GetOk("date_partition_sequence"); ok && target { // target
		s3s.DatePartitionSequence = aws.String(v.(string))
	}

	if v, ok := d.GetOk("date_partition_timezone"); ok && target { // target
		s3s.DatePartitionTimezone = aws.String(v.(string))
	}

	if v, ok := d.GetOk("dict_page_size_limit"); ok { // likely only useful for target
		s3s.DictPageSizeLimit = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.Get("enable_statistics").(bool); ok { // likely only useful for target
		s3s.EnableStatistics = aws.Bool(v)
	}

	if v, ok := d.GetOk("encoding_type"); ok { // likely only useful for target
		s3s.EncodingType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("encryption_mode"); ok && target { // target
		s3s.EncryptionMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("expected_bucket_owner"); ok { // likely only useful for target
		s3s.ExpectedBucketOwner = aws.String(v.(string))
	}

	if v, ok := d.GetOk("external_table_definition"); ok {
		s3s.ExternalTableDefinition = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ignore_header_rows"); ok {
		s3s.IgnoreHeaderRows = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.Get("include_op_for_full_load").(bool); ok { // likely only useful for target
		s3s.IncludeOpForFullLoad = aws.Bool(v)
	}

	if v, ok := d.GetOk("max_file_size"); ok { // likely only useful for target
		s3s.MaxFileSize = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.Get("parquet_timestamp_in_millisecond").(bool); ok && target { // target
		s3s.ParquetTimestampInMillisecond = aws.Bool(v)
	}

	if v, ok := d.GetOk("parquet_version"); ok && target { // target
		s3s.ParquetVersion = aws.String(v.(string))
	}

	if v, ok := d.Get("preserve_transactions").(bool); ok && target { // target
		s3s.PreserveTransactions = aws.Bool(v)
	}

	if v, ok := d.Get("rfc_4180").(bool); ok {
		s3s.Rfc4180 = aws.Bool(v)
	}

	if v, ok := d.GetOk("row_group_length"); ok { // likely only useful for target
		s3s.RowGroupLength = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("server_side_encryption_kms_key_id"); ok && target { // target
		s3s.ServerSideEncryptionKmsKeyId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_access_role_arn"); ok {
		s3s.ServiceAccessRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("timestamp_column_name"); ok { // likely only useful for target
		s3s.TimestampColumnName = aws.String(v.(string))
	}

	if v, ok := d.Get("use_csv_no_sup_value").(bool); ok && target { // target
		s3s.UseCsvNoSupValue = aws.Bool(v)
	}

	if v, ok := d.Get("use_task_start_time_for_full_load_timestamp").(bool); ok { // likely only useful for target
		s3s.UseTaskStartTimeForFullLoadTimestamp = aws.Bool(v)
	}

	return s3s
}
